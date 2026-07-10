package storage

import "time"

type Feed struct {
	ID        int64
	Name      string
	URL       string
	CreatedAt time.Time
}

type Post struct {
	ID          int64
	FeedID      int64
	Title       string
	URL         string
	PublishedAt time.Time
	CreatedAt   time.Time
}
