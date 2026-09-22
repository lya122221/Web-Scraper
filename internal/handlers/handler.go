package handlers

import (
	"context"
	"time"
	"web-scraper/internal/storage"
)

type StorageInterface interface {
	GetFeeds(ctx context.Context) ([]storage.Feed, error)
	AddFeed(ctx context.Context, name string, url string) (int64, error)
	GetPosts(ctx context.Context, limit int) ([]storage.Post, error)
	AddPost(ctx context.Context, feedID int64, title string, url string, publishedAt *time.Time) error
}

type Handler struct {
	s StorageInterface
}

func NewHandler(s StorageInterface) *Handler {
	return &Handler{s}
}
