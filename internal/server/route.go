package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/vulan1999/go-dag/web/pages"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	// Set 10 second timeout for testing graceful shutdown
	time.Sleep(10 * time.Second)
	response := map[string]string{
		"status": "ok",
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(response)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	err := pages.Home().Render(r.Context(), w)

	if err != nil {
		log.Printf("Fail to render page: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
