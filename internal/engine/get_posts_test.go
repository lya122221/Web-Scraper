package engine

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetPosts_PrivateAddressBlocked(t *testing.T) {
	_, err := getPosts(context.Background(), "http://127.0.0.1:8080/rss")
	if err == nil {
		t.Fatal("expected private address error, got nil")
	}

	if !strings.Contains(err.Error(), "private network address") {
		t.Fatalf("expected private address error, got %v", err)
	}
}

func TestGetPosts_Success(t *testing.T) {
	rss := RSSFeed{
		Channel: RSSChannel{
			Title:       "Test Channel",
			Description: "Test Description",
			Items: []RSSItem{
				{Title: "Post 1", Link: "https://example.com/1", PubDate: "Mon, 01 Jan 2024 00:00:00 +0000"},
				{Title: "Post 2", Link: "https://example.com/2", PubDate: "Tue, 02 Jan 2024 00:00:00 +0000"},
			},
		},
	}

	xmlBytes, _ := xml.Marshal(rss)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write(xmlBytes)
	}))
	defer server.Close()

	result, err := getPostsWithClient(context.Background(), server.URL, server.Client())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Channel.Title != "Test Channel" {
		t.Fatalf("expected channel title 'Test Channel', got '%s'", result.Channel.Title)
	}

	if len(result.Channel.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Channel.Items))
	}

	if result.Channel.Items[0].Title != "Post 1" {
		t.Fatalf("expected first item title 'Post 1', got '%s'", result.Channel.Items[0].Title)
	}
}

func TestGetPosts_EmptyFeed(t *testing.T) {
	rss := RSSFeed{
		Channel: RSSChannel{
			Title: "Empty Channel",
		},
	}

	xmlBytes, _ := xml.Marshal(rss)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write(xmlBytes)
	}))
	defer server.Close()

	result, err := getPostsWithClient(context.Background(), server.URL, server.Client())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Channel.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(result.Channel.Items))
	}
}

func TestGetPosts_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := getPostsWithClient(context.Background(), server.URL, server.Client())
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestGetPosts_NotFoundError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := getPostsWithClient(context.Background(), server.URL, server.Client())
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
}

func TestGetPosts_InvalidXML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("this is not xml"))
	}))
	defer server.Close()

	_, err := getPostsWithClient(context.Background(), server.URL, server.Client())
	if err == nil {
		t.Fatal("expected error for invalid XML, got nil")
	}
}

func TestGetPosts_InvalidURL(t *testing.T) {
	_, err := getPosts(context.Background(), "://invalid-url")
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestGetPosts_CanceledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<rss></rss>"))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := getPostsWithClient(ctx, server.URL, server.Client())
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}

func TestGetPosts_UserAgentHeader(t *testing.T) {
	var receivedUA string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		rss := RSSFeed{Channel: RSSChannel{Title: "Test"}}
		xmlBytes, _ := xml.Marshal(rss)
		w.Write(xmlBytes)
	}))
	defer server.Close()

	_, err := getPostsWithClient(context.Background(), server.URL, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedUA != "Mozilla/5.0" {
		t.Fatalf("expected User-Agent 'Mozilla/5.0', got '%s'", receivedUA)
	}
}
