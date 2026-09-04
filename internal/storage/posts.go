package storage

import "time"

func (s *Storage) CreatePostsTable() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS posts (
			id BIGSERIAL PRIMARY KEY,
			feed_id BIGINT NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			url TEXT NOT NULL UNIQUE,
			published_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
	`)

	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) AddPost(feedID int64, title string, url string, publishedAt *time.Time) error {
	_, err := s.db.Exec(`
	INSERT INTO posts (feed_id, title, url, published_at)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (url) DO NOTHING
	`, feedID, title, url, publishedAt)

	return err
}

func (s *Storage) GetPosts(limit int) ([]Post, error) {
	rows, err := s.db.Query(`
	SELECT id, feed_id, title, url, published_at, created_at
	FROM posts
	ORDER BY published_at DESC NULLS LAST
	LIMIT $1
	`, limit)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var posts []Post

	for rows.Next() {
		var p Post

		err := rows.Scan(&p.ID, &p.FeedID, &p.Title, &p.URL, &p.PublishedAt, &p.CreatedAt)
		if err != nil {
			return nil, err
		}

		posts = append(posts, p)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return posts, nil
}
