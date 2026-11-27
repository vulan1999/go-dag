package pipeline

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestHandler(t *testing.T) (*Handler, Repository) {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo, err := NewSQLiteRepository(db)
	if err != nil {
		t.Fatalf("failed to init repository: %v", err)
	}

	return NewHandler(repo), repo
}

func TestHandler_Create_JSON_Success(t *testing.T) {
	h, repo := setupTestHandler(t)

	body := `{
		"name": "test",
		"nodes": [],
		"edges": []
	}`
	req := httptest.NewRequest(http.MethodPost, "/pipelines", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json Content-Type, got %s", got)
	}

	var p Pipeline
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if p.ID == "" {
		t.Fatalf("expected generated pipeline ID, got empty string")
	}
	if p.Name != "test" {
		t.Errorf("expected name %q, got %q", "test", p.Name)
	}

	stored, err := repo.GetByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("expected pipeline to be persisted, got err: %v", err)
	}
	if stored.Name != "test" {
		t.Errorf("expected stored name %q, got %q", "test", stored.Name)
	}
}

func TestHandler_Create_HTMX_Success(t *testing.T) {
	h, _ := setupTestHandler(t)

	form := url.Values{}
	form.Set("name", "HTMX Pipeline")
	form.Set("nodes", `[{"id":"n1","type":"source","config":{}}]`)
	form.Set("edges", `[]`)

	req := httptest.NewRequest(http.MethodPost, "/pipelines", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("expected text/html Content-Type, got %s", ct)
	}
	if !strings.Contains(rec.Body.String(), "HTMX Pipeline") {
		t.Errorf("expected rendered row to contain pipeline name, got %s", rec.Body.String())
	}
}

func TestHandler_Create_InvalidRequest(t *testing.T) {
	h, _ := setupTestHandler(t)

	testRequests := []struct {
		name string
		body string
	}{
		{name: "invalid JSON request", body: `{"name":`},
		{name: "empty body", body: ""},
		{name: "missing pipeline name", body: `{"nodes": []}`},
		{name: "empty pipeline name", body: `{"name": "", "nodes": [], "edges": []}`},
		{name: "whitespace pipeline name", body: `{"name": "   ", "nodes": [], "edges": []}`},
		{name: "multi JSON value", body: `{"name": "A", "nodes": [], "edges": []} {"name": "B", "nodes": [], "edges": []}`},
	}

	for _, tt := range testRequests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/pipelines", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.Create(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		})
	}
}

func TestHandler_List_HTMLAndJSON(t *testing.T) {
	h, repo := setupTestHandler(t)

	_ = repo.Create(context.Background(), &Pipeline{
		ID:    "p1",
		Name:  "Daily ETL",
		Nodes: []Node{},
		Edges: []Edge{},
	})

	// HTML page
	reqHTML := httptest.NewRequest(http.MethodGet, "/pipelines", nil)
	recHTML := httptest.NewRecorder()
	h.List(recHTML, reqHTML)

	if recHTML.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recHTML.Code)
	}
	if !strings.Contains(recHTML.Body.String(), "Daily ETL") {
		t.Errorf("expected HTML page to contain pipeline name")
	}

	// JSON response
	reqJSON := httptest.NewRequest(http.MethodGet, "/pipelines", nil)
	reqJSON.Header.Set("Accept", "application/json")
	recJSON := httptest.NewRecorder()
	h.List(recJSON, reqJSON)

	if recJSON.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recJSON.Code)
	}
	var list []Pipeline
	if err := json.NewDecoder(recJSON.Body).Decode(&list); err != nil {
		t.Fatalf("failed to decode JSON list: %v", err)
	}
	if len(list) != 1 || list[0].ID != "p1" {
		t.Errorf("unexpected JSON list: %+v", list)
	}
}

func TestHandler_GetByID_Update_Delete(t *testing.T) {
	h, repo := setupTestHandler(t)

	_ = repo.Create(context.Background(), &Pipeline{
		ID:    "p1",
		Name:  "Initial",
		Nodes: []Node{},
		Edges: []Edge{},
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /pipelines/{id}", h.GetByID)
	mux.HandleFunc("PUT /pipelines/{id}", h.Update)
	mux.HandleFunc("DELETE /pipelines/{id}", h.Delete)

	// GetByID JSON
	reqGet := httptest.NewRequest(http.MethodGet, "/pipelines/p1", nil)
	recGet := httptest.NewRecorder()
	mux.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200 on GET, got %d", recGet.Code)
	}

	// GetByID HTMX edit form
	reqEdit := httptest.NewRequest(http.MethodGet, "/pipelines/p1", nil)
	reqEdit.Header.Set("HX-Request", "true")
	recEdit := httptest.NewRecorder()
	mux.ServeHTTP(recEdit, reqEdit)
	if recEdit.Code != http.StatusOK || !strings.Contains(recEdit.Body.String(), "<form") {
		t.Fatalf("expected HTMX edit form, got status %d body %s", recEdit.Code, recEdit.Body.String())
	}

	// Update HTMX
	form := url.Values{}
	form.Set("name", "Updated Name")
	form.Set("nodes", `[]`)
	form.Set("edges", `[]`)
	reqPut := httptest.NewRequest(http.MethodPut, "/pipelines/p1", strings.NewReader(form.Encode()))
	reqPut.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqPut.Header.Set("HX-Request", "true")
	recPut := httptest.NewRecorder()
	mux.ServeHTTP(recPut, reqPut)
	if recPut.Code != http.StatusOK || !strings.Contains(recPut.Body.String(), "Updated Name") {
		t.Fatalf("expected updated row, got status %d body %s", recPut.Code, recPut.Body.String())
	}

	// Delete HTMX
	reqDel := httptest.NewRequest(http.MethodDelete, "/pipelines/p1", nil)
	reqDel.Header.Set("HX-Request", "true")
	recDel := httptest.NewRecorder()
	mux.ServeHTTP(recDel, reqDel)
	if recDel.Code != http.StatusOK {
		t.Fatalf("expected 200 on HTMX DELETE, got %d", recDel.Code)
	}

	// Confirm 404 after delete
	reqMissing := httptest.NewRequest(http.MethodGet, "/pipelines/p1", nil)
	recMissing := httptest.NewRecorder()
	mux.ServeHTTP(recMissing, reqMissing)
	if recMissing.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", recMissing.Code)
	}
}
