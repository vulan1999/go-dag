package pipeline

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type CreatePipelineRequest struct {
	Name  string `json:"name"`
	Nodes []Node `json:"nodes"`
	Edge  []Edge `json:"edges"`
}

func CreateHandler(w http.ResponseWriter, r *http.Request) {
	var req CreatePipelineRequest
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Invalid Request body", http.StatusBadRequest)
		return
	}

	// Ensure the request body just contains only one JSON object value
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		http.Error(w, "Request must contain a single JSON value", http.StatusBadRequest)
		return
	}

	// Validate pipeline name: Pipeline name must not be empty
	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "Pipeline name is required", http.StatusBadRequest)
		return
	}

	// Initial new pipeline object
	p := Pipeline{
		ID:    uuid.NewString(),
		Name:  req.Name,
		Nodes: req.Nodes,
		Edges: req.Edge,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(p); err != nil {
		return
	}
}
