package storage

func (s *Storage) CreateFeedsTable() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS feeds (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			url TEXT NOT NULL UNIQUE,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
	`)

	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) AddFeed(name string, url string) (int64, error) {
	var id int64

	err := s.db.QueryRow(`
		INSERT INTO feeds (name, url)
		VALUES ($1, $2)
		RETURNING id
	`, name, url).Scan(&id)

	return id, err
}

func (s *Storage) GetFeeds() ([]Feed, error) {
	rows, err := s.db.Query(`
		SELECT id, name, url, created_at
		FROM feeds
	`)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var feeds []Feed

	for rows.Next() {
		var f Feed

		err := rows.Scan(&f.ID, &f.Name, &f.URL, &f.CreatedAt)
		if err != nil {
			return nil, err
		}

		feeds = append(feeds, f)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return feeds, nil
}
