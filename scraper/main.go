package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"web-scraper/internal/engine"
	"web-scraper/internal/handlers"
	"web-scraper/internal/storage"

	"github.com/joho/godotenv"
)

func ConnectToDB() (*storage.Storage, error) {
	log.Println("Connecting to database...")

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL environment variable is required")
	}

	s, err := storage.NewStorage(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to launch storage: %w", err)
	}

	log.Println("Successful connection to PostgreSQL")

	return s, nil
}

var workerCount int = 10
var jobsCount int = 50

const engineShutdownTimeout = 30 * time.Second

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	s, err := ConnectToDB()
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	wp := engine.NewWorkerPool(workerCount, jobsCount)

	e := engine.NewEngine(wp, s)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	wp.Start(ctx)
	engineErr := make(chan error, 1)
	go func() {
		engineErr <- e.Start(ctx)
	}()

	h := handlers.NewHandler(s)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/feeds", h.FeedHandler)
	mux.HandleFunc("/api/posts", h.PostsHandler)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	serverErr := make(chan error, 1)
	go func() {
		log.Print("Server started at port 8080")
		serverErr <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Server error: %v", err)
		}
		stop()
	}

	log.Println("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Failed to shutdown server: %v", err)
	}
	cancel()

	engineShutdownCtx, cancelEngine := context.WithTimeout(context.Background(), engineShutdownTimeout)
	defer cancelEngine()

	select {
	case err := <-engineErr:
		if err != nil {
			log.Printf("Engine stopped with error: %v", err)
		}
	case <-engineShutdownCtx.Done():
		log.Printf("Failed to shutdown engine: %v", engineShutdownCtx.Err())
	}
}
