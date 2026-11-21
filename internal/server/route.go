package server

import (
	"encoding/json"
	"net/http"
	"time"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(10 * time.Second)
	response := map[string]string{
		"status": "ok",
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(response)
}
