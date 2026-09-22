package engine

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
	"web-scraper/internal/storage"
)

type mockEngineStorage struct {
	mu        sync.Mutex
	feeds     []storage.Feed
	feedsErr  error
	posts     []addPostCall
	postErr   error
	postAdded chan struct{}
}

type addPostCall struct {
	FeedID      int64
	Title       string
	URL         string
	PublishedAt *time.Time
}

func (m *mockEngineStorage) GetFeeds(ctx context.Context) ([]storage.Feed, error) {
	return m.feeds, m.feedsErr
}

func (m *mockEngineStorage) AddPost(ctx context.Context, feedID int64, title string, url string, publishedAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.posts = append(m.posts, addPostCall{feedID, title, url, publishedAt})
	if m.postAdded != nil {
		m.postAdded <- struct{}{}
	}
	return m.postErr
}

func (m *mockEngineStorage) getPosts() []addPostCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]addPostCall, len(m.posts))
	copy(result, m.posts)
	return result
}

func waitForSignals(t *testing.T, signals <-chan struct{}, count int) {
	t.Helper()

	for i := 0; i < count; i++ {
		select {
		case <-signals:
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for signal %d", i+1)
		}
	}
}

func TestNewEngine(t *testing.T) {
	wp := NewWorkerPool(1, 1)
	mock := &mockEngineStorage{}

	e := NewEngine(wp, mock)
	if e == nil {
		t.Fatal("expected non-nil engine")
	}

	if e.wp != wp {
		t.Fatal("expected worker pool to match")
	}

	if e.fetch == nil {
		t.Fatal("expected fetch function to be set")
	}
}

func TestEngine_Start_ImmediateCancel(t *testing.T) {
	wp := NewWorkerPool(2, 10)
	mock := &mockEngineStorage{
		feeds: []storage.Feed{},
	}

	e := NewEngine(wp, mock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	wp.Start(ctx)

	err := e.Start(ctx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestEngine_Start_GetFeedsError(t *testing.T) {
	wp := NewWorkerPool(2, 10)
	mock := &mockEngineStorage{
		feedsErr: errors.New("db error"),
	}

	e := NewEngine(wp, mock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	wp.Start(ctx)

	err := e.Start(ctx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestEngine_CheckAllAndAddToDB_Success(t *testing.T) {
	wp := NewWorkerPool(2, 10)
	mock := &mockEngineStorage{postAdded: make(chan struct{}, 2)}

	e := NewEngine(wp, mock)

	e.fetch = func(ctx context.Context, url string) (*RSSFeed, error) {
		return &RSSFeed{
			Channel: RSSChannel{
				Title: "Test",
				Items: []RSSItem{
					{Title: "Post 1", Link: "https://example.com/1", PubDate: "Mon, 01 Jan 2024 00:00:00 +0000"},
					{Title: "Post 2", Link: "https://example.com/2", PubDate: "Tue, 02 Jan 2024 00:00:00 GMT"},
				},
			},
		}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp.Start(ctx)

	feeds := []storage.Feed{
		{ID: 1, Name: "Feed 1", URL: "https://example.com/rss"},
	}

	e.checkAllAndAddToDB(ctx, feeds)

	waitForSignals(t, mock.postAdded, 2)

	posts := mock.getPosts()
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts added, got %d", len(posts))
	}

	if posts[0].FeedID != 1 {
		t.Fatalf("expected feed_id 1, got %d", posts[0].FeedID)
	}

	if posts[0].Title != "Post 1" {
		t.Fatalf("expected title 'Post 1', got '%s'", posts[0].Title)
	}
}

func TestEngine_CheckAllAndAddToDB_FetchError(t *testing.T) {
	wp := NewWorkerPool(2, 10)
	mock := &mockEngineStorage{}
	fetchCalled := make(chan struct{})

	e := NewEngine(wp, mock)

	e.fetch = func(ctx context.Context, url string) (*RSSFeed, error) {
		close(fetchCalled)
		return nil, errors.New("network error")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp.Start(ctx)

	feeds := []storage.Feed{
		{ID: 1, Name: "Feed 1", URL: "https://example.com/rss"},
	}

	e.checkAllAndAddToDB(ctx, feeds)

	waitForSignals(t, fetchCalled, 1)

	posts := mock.getPosts()
	if len(posts) != 0 {
		t.Fatalf("expected 0 posts on fetch error, got %d", len(posts))
	}
}

func TestEngine_CheckAllAndAddToDB_AddPostError(t *testing.T) {
	wp := NewWorkerPool(2, 10)
	mock := &mockEngineStorage{
		postErr:   errors.New("db insert error"),
		postAdded: make(chan struct{}, 1),
	}

	e := NewEngine(wp, mock)

	e.fetch = func(ctx context.Context, url string) (*RSSFeed, error) {
		return &RSSFeed{
			Channel: RSSChannel{
				Items: []RSSItem{
					{Title: "Post 1", Link: "https://example.com/1", PubDate: "Mon, 01 Jan 2024 00:00:00 +0000"},
				},
			},
		}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp.Start(ctx)

	feeds := []storage.Feed{
		{ID: 1, Name: "Feed 1", URL: "https://example.com/rss"},
	}

	e.checkAllAndAddToDB(ctx, feeds)

	waitForSignals(t, mock.postAdded, 1)

	posts := mock.getPosts()
	if len(posts) != 1 {
		t.Fatalf("expected 1 post call (even with error), got %d", len(posts))
	}
}

func TestEngine_CheckAllAndAddToDB_InvalidPubDate(t *testing.T) {
	wp := NewWorkerPool(2, 10)
	mock := &mockEngineStorage{postAdded: make(chan struct{}, 1)}

	e := NewEngine(wp, mock)

	e.fetch = func(ctx context.Context, url string) (*RSSFeed, error) {
		return &RSSFeed{
			Channel: RSSChannel{
				Items: []RSSItem{
					{Title: "Post Invalid Date", Link: "https://example.com/1", PubDate: "not-a-date"},
				},
			},
		}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp.Start(ctx)

	feeds := []storage.Feed{
		{ID: 1, Name: "Feed 1", URL: "https://example.com/rss"},
	}

	e.checkAllAndAddToDB(ctx, feeds)

	waitForSignals(t, mock.postAdded, 1)

	posts := mock.getPosts()
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}

	if posts[0].PublishedAt != nil {
		t.Fatal("expected nil PublishedAt for invalid date")
	}
}

func TestEngine_CheckAllAndAddToDB_MultipleFeeds(t *testing.T) {
	wp := NewWorkerPool(3, 20)
	mock := &mockEngineStorage{postAdded: make(chan struct{}, 3)}

	e := NewEngine(wp, mock)

	e.fetch = func(ctx context.Context, url string) (*RSSFeed, error) {
		return &RSSFeed{
			Channel: RSSChannel{
				Items: []RSSItem{
					{Title: "Post from " + url, Link: url + "/post", PubDate: "Mon, 01 Jan 2024 00:00:00 +0000"},
				},
			},
		}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp.Start(ctx)

	feeds := []storage.Feed{
		{ID: 1, Name: "Feed 1", URL: "https://a.com/rss"},
		{ID: 2, Name: "Feed 2", URL: "https://b.com/rss"},
		{ID: 3, Name: "Feed 3", URL: "https://c.com/rss"},
	}

	e.checkAllAndAddToDB(ctx, feeds)

	waitForSignals(t, mock.postAdded, 3)

	posts := mock.getPosts()
	if len(posts) != 3 {
		t.Fatalf("expected 3 posts from 3 feeds, got %d", len(posts))
	}
}

func TestEngine_CheckAllAndAddToDB_EmptyFeeds(t *testing.T) {
	wp := NewWorkerPool(2, 10)
	mock := &mockEngineStorage{}

	e := NewEngine(wp, mock)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp.Start(ctx)

	e.checkAllAndAddToDB(ctx, []storage.Feed{})

	posts := mock.getPosts()
	if len(posts) != 0 {
		t.Fatalf("expected 0 posts for empty feeds, got %d", len(posts))
	}
}

func TestEngine_CheckAllAndAddToDB_RFC1123Date(t *testing.T) {
	wp := NewWorkerPool(2, 10)
	mock := &mockEngineStorage{postAdded: make(chan struct{}, 1)}

	e := NewEngine(wp, mock)

	e.fetch = func(ctx context.Context, url string) (*RSSFeed, error) {
		return &RSSFeed{
			Channel: RSSChannel{
				Items: []RSSItem{
					{Title: "RFC1123 Post", Link: "https://example.com/1", PubDate: "Mon, 01 Jan 2024 00:00:00 GMT"},
				},
			},
		}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp.Start(ctx)

	feeds := []storage.Feed{
		{ID: 1, Name: "Feed 1", URL: "https://example.com/rss"},
	}

	e.checkAllAndAddToDB(ctx, feeds)

	waitForSignals(t, mock.postAdded, 1)

	posts := mock.getPosts()
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}

	if posts[0].PublishedAt == nil {
		t.Fatal("expected non-nil PublishedAt for valid RFC1123 date")
	}
}

func TestEngine_Start_WithFeedsProcessing(t *testing.T) {
	wp := NewWorkerPool(2, 10)
	mock := &mockEngineStorage{
		feeds: []storage.Feed{
			{ID: 1, Name: "Feed 1", URL: "https://example.com/rss"},
		},
		postAdded: make(chan struct{}, 1),
	}

	e := NewEngine(wp, mock)

	e.fetch = func(ctx context.Context, url string) (*RSSFeed, error) {
		return &RSSFeed{
			Channel: RSSChannel{
				Items: []RSSItem{
					{Title: "Startup Post", Link: "https://example.com/1", PubDate: "Mon, 01 Jan 2024 00:00:00 +0000"},
				},
			},
		}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	wp.Start(ctx)

	go func() {
		<-mock.postAdded
		cancel()
	}()

	err := e.Start(ctx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	posts := mock.getPosts()
	if len(posts) != 1 {
		t.Fatalf("expected 1 post from startup run, got %d", len(posts))
	}
}

func TestEngine_Start_WaitsForRunningJob(t *testing.T) {
	wp := NewWorkerPool(1, 1)
	mock := &mockEngineStorage{
		feeds: []storage.Feed{
			{ID: 1, Name: "Feed 1", URL: "https://example.com/rss"},
		},
		postAdded: make(chan struct{}, 1),
	}

	e := NewEngine(wp, mock)
	started := make(chan struct{})
	release := make(chan struct{})
	e.fetch = func(ctx context.Context, url string) (*RSSFeed, error) {
		close(started)
		<-release
		return &RSSFeed{
			Channel: RSSChannel{
				Items: []RSSItem{
					{Title: "Post", Link: "https://example.com/1", PubDate: "Mon, 01 Jan 2024 00:00:00 +0000"},
				},
			},
		}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	wp.Start(ctx)

	engineDone := make(chan error, 1)
	go func() {
		engineDone <- e.Start(ctx)
	}()

	<-started
	cancel()
	close(release)

	if err := <-engineDone; err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	posts := mock.getPosts()
	if len(posts) != 1 {
		t.Fatalf("expected running job to add 1 post, got %d", len(posts))
	}
}
