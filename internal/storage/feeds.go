package storage

import (
	"database/sql"
	"fmt"
)

func createFeedsTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS feeds (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			url TEXT NOT NULL UNIQUE,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
	`)

	if err != nil {
		return fmt.Errorf("(Error)", err.Error())
	}

	return nil
}
