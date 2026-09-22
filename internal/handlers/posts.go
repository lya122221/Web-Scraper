package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

func (h *Handler) PostsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	limit := 50

	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		limitInt, err := strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "Limit must be a number", http.StatusBadRequest)
			return
		}
		limit = limitInt
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	posts, err := h.s.GetPosts(r.Context(), limit)
	if err != nil {
		log.Printf("Failed to get posts: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(posts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
