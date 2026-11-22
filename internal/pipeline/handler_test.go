package pipeline

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateHandler_Success(t *testing.T) {
	body := `{
		"name": "test",
		"nodes": [],
		"edges": []
	}`

	req := httptest.NewRequest(http.MethodPost, "/pipelines", strings.NewReader(body))

	rec := httptest.NewRecorder()

	CreateHandler(rec, req)

	// Validate http status code
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	// validate request header
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json Content-Type, got %s", got)
	}

	// validate json result
	var pipeline Pipeline
	if err := json.NewDecoder(rec.Body).Decode(&pipeline); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if pipeline.ID == "" {
		t.Fatalf("expected uuid generated pipeline ID, got empty string")
	}

	if pipeline.Name != "test" {
		t.Errorf("expected name %s, got %q", "test", pipeline.Name)
	}

	if pipeline.Nodes == nil {
		t.Errorf("expected nodes is [], got %q", pipeline.Nodes)
	}

	if pipeline.Edges == nil {
		t.Errorf("expected edges is [], got %q", pipeline.Edges)
	}
}

func TestCreateHandler_InvalidRequest(t *testing.T) {
	test_requests := []struct {
		name string
		body string
	}{
		{
			name: "invalid JSON request",
			body: `{"name":`,
		},
		{
			name: "empty body",
			body: "",
		},
		{
			name: "missing pipeline name",
			body: `{"nodes": []}`,
		},
		{
			name: "empty pipeline name",
			body: `{"name": "", "nodes": [], "edges": []}`,
		},
		{
			name: "pipeline name contain all whitespace characters",
			body: `{"name": "   ", "nodes": [], "edges": []}`,
		},
		{
			name: "multi JSON value in body",
			body: `{"name": "A", "nodes": [], "edges": []} {"name": "B", "nodes": [], "edges": []}`,
		},
	}

	for _, tt := range test_requests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/pipelines", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			CreateHandler(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected http status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		})
	}
}
