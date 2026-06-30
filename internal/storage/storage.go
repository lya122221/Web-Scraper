package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Storage struct {
	db *sql.DB
}

func NewStorage(dsn string) (*Storage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("(Error)", err.Error())
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("(Error)", err.Error())
	}

	err = createFeedsTables(db)
	if err != nil {
		return nil, fmt.Errorf("(Error)", err.Error())
	}

	err = createPostsTables(db)
	if err != nil {
		return nil, fmt.Errorf("(Error)", err.Error())
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	err := s.db.Close()
	if err != nil {
		return fmt.Errorf("(Error)", err.Error())
	}
	return nil
}
