package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"web-scraper/internal/storage"
)

type Feed struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (h *Handler) methodGet(w http.ResponseWriter, r *http.Request) {
	feeds, err := h.s.GetFeeds(r.Context())

	if err != nil {
		log.Printf("Error get feeds: %v", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(feeds)
	if err != nil {
		log.Printf("Error encode json: %v", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) methodPost(w http.ResponseWriter, r *http.Request) {
	var feed Feed
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&feed)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if feed.Name == "" || feed.URL == "" {
		http.Error(w, "Name and url are required", http.StatusBadRequest)
		return
	}

	parsedURL, err := url.ParseRequestURI(feed.URL)
	if err != nil || parsedURL.Hostname() == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		http.Error(w, "Invalid url format", http.StatusBadRequest)
		return
	}

	hostname := strings.ToLower(parsedURL.Hostname())
	if hostname == "localhost" {
		http.Error(w, "Private network urls are not allowed", http.StatusBadRequest)
		return
	}

	if ip, err := netip.ParseAddr(hostname); err == nil {
		if !ip.IsGlobalUnicast() || ip.IsPrivate() {
			http.Error(w, "Private network urls are not allowed", http.StatusBadRequest)
			return
		}
	}

	id, err := h.s.AddFeed(r.Context(), feed.Name, feed.URL)
	if err != nil {
		if errors.Is(err, storage.ErrFeedAlreadyExists) {
			http.Error(w, "Feed already exists", http.StatusConflict)
			return
		}

		log.Printf("Failed to add feed: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Added new feed, id: %d", id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(map[string]int64{
		"id": id,
	})
	if err != nil {
		log.Printf("Error encode json: %v", err)
		return
	}
}

func (h *Handler) FeedHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.methodGet(w, r)
	case http.MethodPost:
		h.methodPost(w, r)
	default:
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
}
