package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
)

func newMockStorage(t *testing.T) (*Storage, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	return &Storage{db: db}, mock
}

func TestStorage_AddFeed(t *testing.T) {
	s, mock := newMockStorage(t)

	mock.ExpectQuery("INSERT INTO feeds").
		WithArgs("Test feed", "https://example.com/rss").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))

	id, err := s.AddFeed(context.Background(), "Test feed", "https://example.com/rss")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if id != 42 {
		t.Fatalf("expected id 42, got %d", id)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStorage_AddFeedDuplicate(t *testing.T) {
	s, mock := newMockStorage(t)

	mock.ExpectQuery("INSERT INTO feeds").
		WithArgs("Test feed", "https://example.com/rss").
		WillReturnError(&pgconn.PgError{Code: "23505"})

	_, err := s.AddFeed(context.Background(), "Test feed", "https://example.com/rss")
	if !errors.Is(err, ErrFeedAlreadyExists) {
		t.Fatalf("expected ErrFeedAlreadyExists, got %v", err)
	}
}

func TestStorage_GetFeeds(t *testing.T) {
	s, mock := newMockStorage(t)
	now := time.Now()

	mock.ExpectQuery("SELECT id, name, url, created_at").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "url", "created_at"}).
			AddRow(1, "Test feed", "https://example.com/rss", now))

	feeds, err := s.GetFeeds(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(feeds) != 1 {
		t.Fatalf("expected 1 feed, got %d", len(feeds))
	}

	if feeds[0].Name != "Test feed" {
		t.Fatalf("expected feed name 'Test feed', got %q", feeds[0].Name)
	}
}

func TestStorage_AddPost(t *testing.T) {
	s, mock := newMockStorage(t)
	publishedAt := time.Now()

	mock.ExpectExec("INSERT INTO posts").
		WithArgs(int64(1), "Test post", "https://example.com/post", &publishedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.AddPost(context.Background(), 1, "Test post", "https://example.com/post", &publishedAt)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestStorage_GetPosts(t *testing.T) {
	s, mock := newMockStorage(t)
	now := time.Now()

	mock.ExpectQuery("ORDER BY created_at DESC").
		WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "feed_id", "title", "url", "published_at", "created_at"}).
			AddRow(1, 2, "Test post", "https://example.com/post", now, now))

	posts, err := s.GetPosts(context.Background(), 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}

	if posts[0].FeedID != 2 {
		t.Fatalf("expected feed id 2, got %d", posts[0].FeedID)
	}
}

func TestStorage_QueryError(t *testing.T) {
	s, mock := newMockStorage(t)
	expectedErr := errors.New("database error")

	mock.ExpectQuery("SELECT id, name, url, created_at").WillReturnError(expectedErr)

	_, err := s.GetFeeds(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected database error, got %v", err)
	}
}
