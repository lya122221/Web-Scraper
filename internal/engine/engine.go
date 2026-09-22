package engine

import (
	"context"
	"log"
	"time"
	"web-scraper/internal/storage"
)

const jobTimeout = 30 * time.Second

type EngineStorage interface {
	GetFeeds(ctx context.Context) ([]storage.Feed, error)
	AddPost(ctx context.Context, feedID int64, title string, url string, publishedAt *time.Time) error
}

type fetchFunc func(ctx context.Context, url string) (*RSSFeed, error)

type Engine struct {
	wp    *WorkerPool
	s     EngineStorage
	fetch fetchFunc
}

func NewEngine(wp *WorkerPool, s EngineStorage) *Engine {
	return &Engine{
		wp:    wp,
		s:     s,
		fetch: getPosts,
	}
}

func (e *Engine) checkAllAndAddToDB(ctx context.Context, feeds []storage.Feed) {
	for _, feed := range feeds {
		err := e.wp.Submit(ctx,
			func() {
				jobCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), jobTimeout)
				defer cancel()

				rssFeed, err := e.fetch(jobCtx, feed.URL)
				if err != nil {
					log.Printf("Failed to get posts from %s: %v", feed.URL, err)
					return
				}

				for _, item := range rssFeed.Channel.Items {
					var pubAtPtr *time.Time

					pubAt, err := time.Parse(time.RFC1123Z, item.PubDate)
					if err != nil {
						pubAt, err = time.Parse(time.RFC1123, item.PubDate)
						if err != nil {
							log.Printf("Failed to parse published date: %v", err)
							pubAtPtr = nil
						} else {
							pubAtPtr = &pubAt
						}
					} else {
						pubAtPtr = &pubAt
					}

					err = e.s.AddPost(jobCtx, feed.ID, item.Title, item.Link, pubAtPtr)
					if err != nil {
						log.Printf("Failed to add post: %v", err)
						continue
					}
				}
			})
		if err != nil {
			log.Printf("Failed to submit job: %v", err)
			return
		}
	}
}

func (e *Engine) Start(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Minute)

	defer func() {
		ticker.Stop()
		e.wp.Stop()
	}()

	run := func() {
		feeds, err := e.s.GetFeeds(ctx)
		if err != nil {
			log.Printf("Failed to get feeds: %v", err)
			return
		}

		e.checkAllAndAddToDB(ctx, feeds)
	}

	run()

	for {
		select {
		case <-ticker.C:
			run()
		case <-ctx.Done():
			return nil
		}
	}
}
