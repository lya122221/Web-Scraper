package storage

import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Storage struct {
	db *sql.DB
}

func NewStorage(dsn string) (*Storage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	s := Storage{db: db}

	err = s.CreateFeedsTable()
	if err != nil {
		return nil, err
	}

	err = s.CreatePostsTable()
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}
