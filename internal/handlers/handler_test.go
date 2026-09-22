package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"web-scraper/internal/storage"
)

type mockStorage struct {
	feeds    []storage.Feed
	feedsErr error

	addFeedID  int64
	addFeedErr error

	posts      []storage.Post
	postsErr   error
	postsLimit int

	addPostErr error
}

func (m *mockStorage) GetFeeds(ctx context.Context) ([]storage.Feed, error) {
	return m.feeds, m.feedsErr
}

func (m *mockStorage) AddFeed(ctx context.Context, name string, url string) (int64, error) {
	return m.addFeedID, m.addFeedErr
}

func (m *mockStorage) GetPosts(ctx context.Context, limit int) ([]storage.Post, error) {
	m.postsLimit = limit
	return m.posts, m.postsErr
}

func (m *mockStorage) AddPost(ctx context.Context, feedID int64, title string, url string, publishedAt *time.Time) error {
	return m.addPostErr
}

func TestNewHandler(t *testing.T) {
	mock := &mockStorage{}
	h := NewHandler(mock)

	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestFeedHandler_GET_Success(t *testing.T) {
	now := time.Now()
	mock := &mockStorage{
		feeds: []storage.Feed{
			{ID: 1, Name: "Test Feed", URL: "https://example.com/rss", CreatedAt: now},
		},
	}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/feeds", nil)
	w := httptest.NewRecorder()

	h.FeedHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %s", contentType)
	}

	var feeds []storage.Feed
	err := json.NewDecoder(resp.Body).Decode(&feeds)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(feeds) != 1 {
		t.Fatalf("expected 1 feed, got %d", len(feeds))
	}

	if feeds[0].Name != "Test Feed" {
		t.Fatalf("expected feed name 'Test Feed', got '%s'", feeds[0].Name)
	}
}

func TestFeedHandler_GET_EmptyFeeds(t *testing.T) {
	mock := &mockStorage{feeds: nil}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/feeds", nil)
	w := httptest.NewRecorder()

	h.FeedHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestFeedHandler_GET_StorageError(t *testing.T) {
	mock := &mockStorage{feedsErr: errors.New("db error")}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/feeds", nil)
	w := httptest.NewRecorder()

	h.FeedHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", resp.StatusCode)
	}
}

func TestFeedHandler_POST_Success(t *testing.T) {
	mock := &mockStorage{addFeedID: 42}
	h := NewHandler(mock)

	body := `{"name": "New Feed", "url": "https://example.com/rss"}`
	req := httptest.NewRequest(http.MethodPost, "/api/feeds", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.FeedHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	var result map[string]int64
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result["id"] != 42 {
		t.Fatalf("expected id 42, got %d", result["id"])
	}
}

func TestFeedHandler_POST_InvalidJSON(t *testing.T) {
	mock := &mockStorage{}
	h := NewHandler(mock)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/feeds", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.FeedHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestFeedHandler_POST_EmptyName(t *testing.T) {
	mock := &mockStorage{}
	h := NewHandler(mock)

	body := `{"name": "", "url": "https://example.com/rss"}`
	req := httptest.NewRequest(http.MethodPost, "/api/feeds", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.FeedHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestFeedHandler_POST_EmptyURL(t *testing.T) {
	mock := &mockStorage{}
	h := NewHandler(mock)

	body := `{"name": "Feed", "url": ""}`
	req := httptest.NewRequest(http.MethodPost, "/api/feeds", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.FeedHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestFeedHandler_POST_InvalidURL(t *testing.T) {
	mock := &mockStorage{}
	h := NewHandler(mock)

	body := `{"name": "Feed", "url": "not-a-url"}`
	req := httptest.NewRequest(http.MethodPost, "/api/feeds", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.FeedHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestFeedHandler_POST_InvalidScheme(t *testing.T) {
	mock := &mockStorage{}
	h := NewHandler(mock)

	body := `{"name": "Feed", "url": "ftp://example.com/rss"}`
	req := httptest.NewRequest(http.MethodPost, "/api/feeds", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.FeedHandler(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Result().StatusCode)
	}
}

func TestFeedHandler_POST_PrivateURL(t *testing.T) {
	mock := &mockStorage{}
	h := NewHandler(mock)

	body := `{"name": "Private", "url": "http://127.0.0.1/rss"}`
	req := httptest.NewRequest(http.MethodPost, "/api/feeds", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.FeedHandler(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Result().StatusCode)
	}
}

func TestFeedHandler_POST_UnknownField(t *testing.T) {
	mock := &mockStorage{}
	h := NewHandler(mock)

	body := `{"name": "Feed", "url": "https://example.com/rss", "unknown": true}`
	req := httptest.NewRequest(http.MethodPost, "/api/feeds", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.FeedHandler(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Result().StatusCode)
	}
}

func TestFeedHandler_POST_Duplicate(t *testing.T) {
	mock := &mockStorage{addFeedErr: storage.ErrFeedAlreadyExists}
	h := NewHandler(mock)

	body := `{"name": "Feed", "url": "https://example.com/rss"}`
	req := httptest.NewRequest(http.MethodPost, "/api/feeds", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.FeedHandler(w, req)

	if w.Result().StatusCode != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", w.Result().StatusCode)
	}
}

func TestFeedHandler_POST_StorageError(t *testing.T) {
	mock := &mockStorage{addFeedErr: errors.New("db insert error")}
	h := NewHandler(mock)

	body := `{"name": "Feed", "url": "https://example.com/rss"}`
	req := httptest.NewRequest(http.MethodPost, "/api/feeds", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.FeedHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", resp.StatusCode)
	}
}

func TestFeedHandler_InvalidMethod_PUT(t *testing.T) {
	mock := &mockStorage{}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodPut, "/api/feeds", nil)
	w := httptest.NewRecorder()

	h.FeedHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", resp.StatusCode)
	}
}

func TestFeedHandler_InvalidMethod_DELETE(t *testing.T) {
	mock := &mockStorage{}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodDelete, "/api/feeds", nil)
	w := httptest.NewRecorder()

	h.FeedHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", resp.StatusCode)
	}
}

func TestPostsHandler_GET_Success(t *testing.T) {
	now := time.Now()
	mock := &mockStorage{
		posts: []storage.Post{
			{ID: 1, FeedID: 1, Title: "Test Post", URL: "https://example.com/post", PublishedAt: &now, CreatedAt: now},
		},
	}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	w := httptest.NewRecorder()

	h.PostsHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %s", contentType)
	}

	var posts []storage.Post
	err := json.NewDecoder(resp.Body).Decode(&posts)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
}

func TestPostsHandler_GET_WithLimit(t *testing.T) {
	mock := &mockStorage{posts: []storage.Post{}}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/posts?limit=10", nil)
	w := httptest.NewRecorder()

	h.PostsHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	if mock.postsLimit != 10 {
		t.Fatalf("expected limit 10, got %d", mock.postsLimit)
	}
}

func TestPostsHandler_GET_InvalidLimit(t *testing.T) {
	mock := &mockStorage{}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/posts?limit=abc", nil)
	w := httptest.NewRecorder()

	h.PostsHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestPostsHandler_GET_NegativeLimit(t *testing.T) {
	mock := &mockStorage{posts: []storage.Post{}}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/posts?limit=-5", nil)
	w := httptest.NewRecorder()

	h.PostsHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestPostsHandler_GET_ZeroLimit(t *testing.T) {
	mock := &mockStorage{posts: []storage.Post{}}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/posts?limit=0", nil)
	w := httptest.NewRecorder()

	h.PostsHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestPostsHandler_GET_LimitOver100(t *testing.T) {
	mock := &mockStorage{posts: []storage.Post{}}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/posts?limit=200", nil)
	w := httptest.NewRecorder()

	h.PostsHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestPostsHandler_GET_StorageError(t *testing.T) {
	mock := &mockStorage{postsErr: errors.New("db error")}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	w := httptest.NewRecorder()

	h.PostsHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", resp.StatusCode)
	}

	if strings.Contains(w.Body.String(), "db error") {
		t.Fatal("response must not expose database error")
	}
}

func TestPostsHandler_InvalidMethod_POST(t *testing.T) {
	mock := &mockStorage{}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/posts", nil)
	w := httptest.NewRecorder()

	h.PostsHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", resp.StatusCode)
	}
}

func TestPostsHandler_GET_NoLimit(t *testing.T) {
	mock := &mockStorage{posts: []storage.Post{}}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	w := httptest.NewRecorder()

	h.PostsHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}
