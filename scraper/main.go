package main

import (
	"log"
	"os"
	"web-scraper/internal/storage"
)

func main() {

	log.Println("Connecting to database...")

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Println("Connecting to default storage")
		dsn = "postgres://postgres:qwerty@localhost:5432/rss_aggregator?sslmode=disable"
	}

	s, err := storage.NewStorage(dsn)
	if err != nil {
		log.Fatalf("(Error) failed to launch storage: %v", err)
	}
	defer s.Close()

	log.Println("Successful connection to PostgreSQL")
}
