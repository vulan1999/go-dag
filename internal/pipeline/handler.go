package pipeline

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/vulan1999/go-dag/web/components"
	"github.com/vulan1999/go-dag/web/pages"
)

type CreatePipelineRequest struct {
	Name  string `json:"name"`
	Nodes []Node `json:"nodes"`
	Edge  []Edge `json:"edges"`
}

type Handler struct {
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

func toPipelineView(p *Pipeline) components.PipelineView {
	nodes := p.Nodes
	if nodes == nil {
		nodes = []Node{}
	}
	edges := p.Edges
	if edges == nil {
		edges = []Edge{}
	}

	nodesBytes, err := json.MarshalIndent(nodes, "", "  ")
	if err != nil {
		nodesBytes = []byte("[]")
	}
	edgesBytes, err := json.MarshalIndent(edges, "", "  ")
	if err != nil {
		edgesBytes = []byte("[]")
	}

	return components.PipelineView{
		ID:        p.ID,
		Name:      p.Name,
		NodesJSON: string(nodesBytes),
		EdgesJSON: string(edgesBytes),
		NodeCount: len(nodes),
		EdgeCount: len(edges),
	}
}

func parsePipelinePayload(r *http.Request) (CreatePipelineRequest, error) {
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/x-www-form-urlencoded") || (isHTMX(r) && !strings.HasPrefix(ct, "application/json")) {
		if err := r.ParseForm(); err != nil {
			return CreatePipelineRequest{}, errors.New("Invalid form data")
		}
		var nodes []Node
		if rawNodes := strings.TrimSpace(r.FormValue("nodes")); rawNodes != "" {
			if err := json.Unmarshal([]byte(rawNodes), &nodes); err != nil {
				return CreatePipelineRequest{}, errors.New("Nodes must be a valid JSON array")
			}
		}
		var edges []Edge
		if rawEdges := strings.TrimSpace(r.FormValue("edges")); rawEdges != "" {
			if err := json.Unmarshal([]byte(rawEdges), &edges); err != nil {
				return CreatePipelineRequest{}, errors.New("Edges must be a valid JSON array")
			}
		}
		return CreatePipelineRequest{
			Name:  r.FormValue("name"),
			Nodes: nodes,
			Edge:  edges,
		}, nil
	}

	var req CreatePipelineRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		return CreatePipelineRequest{}, errors.New("Invalid Request body")
	}

	// Ensure the request body contains only one JSON object value
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return CreatePipelineRequest{}, errors.New("Request must contain a single JSON value")
	}

	return req, nil
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	pipelines, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, "Failed to list pipelines", http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(pipelines)
		return
	}

	views := make([]components.PipelineView, 0, len(pipelines))
	for _, p := range pipelines {
		views = append(views, toPipelineView(p))
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.Pipelines(views).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render pipelines page", http.StatusInternalServerError)
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := parsePipelinePayload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		http.Error(w, "Pipeline name is required", http.StatusBadRequest)
		return
	}

	nodes := req.Nodes
	if nodes == nil {
		nodes = []Node{}
	}
	edges := req.Edge
	if edges == nil {
		edges = []Edge{}
	}

	p := Pipeline{
		ID:    uuid.NewString(),
		Name:  name,
		Nodes: nodes,
		Edges: edges,
	}

	if err := h.repo.Create(r.Context(), &p); err != nil {
		http.Error(w, "Failed to create pipeline", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = components.PipelineRow(toPipelineView(&p)).Render(r.Context(), w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(p)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if strings.TrimSpace(id) == "" {
		http.Error(w, "Pipeline ID is required", http.StatusBadRequest)
		return
	}

	p, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "Pipeline not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get pipeline", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		view := toPipelineView(p)
		if r.URL.Query().Get("mode") == "view" {
			_ = components.PipelineRow(view).Render(r.Context(), w)
			return
		}
		_ = components.PipelineEditRow(view, "").Render(r.Context(), w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(p)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if strings.TrimSpace(id) == "" {
		http.Error(w, "Pipeline ID is required", http.StatusBadRequest)
		return
	}

	req, err := parsePipelinePayload(r)
	if err != nil {
		if isHTMX(r) {
			view := components.PipelineView{
				ID:        id,
				Name:      r.FormValue("name"),
				NodesJSON: r.FormValue("nodes"),
				EdgesJSON: r.FormValue("edges"),
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_ = components.PipelineEditRow(view, err.Error()).Render(r.Context(), w)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		if isHTMX(r) {
			view := components.PipelineView{
				ID:        id,
				Name:      req.Name,
				NodesJSON: r.FormValue("nodes"),
				EdgesJSON: r.FormValue("edges"),
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_ = components.PipelineEditRow(view, "Pipeline name is required").Render(r.Context(), w)
			return
		}
		http.Error(w, "Pipeline name is required", http.StatusBadRequest)
		return
	}

	nodes := req.Nodes
	if nodes == nil {
		nodes = []Node{}
	}
	edges := req.Edge
	if edges == nil {
		edges = []Edge{}
	}

	p := Pipeline{
		ID:    id,
		Name:  name,
		Nodes: nodes,
		Edges: edges,
	}

	if err := h.repo.Update(r.Context(), &p); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "Pipeline not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to update pipeline", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = components.PipelineRow(toPipelineView(&p)).Render(r.Context(), w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(p)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if strings.TrimSpace(id) == "" {
		http.Error(w, "Pipeline ID is required", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "Pipeline not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete pipeline", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
