package storage

import (
	"database/sql"
	"fmt"
)

func createPostsTables(db *sql.DB) error {
	_, err := db.Exec(`
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
		return fmt.Errorf("(Error)", err.Error())
	}

	return nil
}
