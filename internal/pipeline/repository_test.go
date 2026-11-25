package pipeline

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) (*SQLiteRepository, func()) {
	t.Helper()

	// Use an in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite db: %v", err)
	}

	repo, err := NewSQLiteRepository(db)
	if err != nil {
		db.Close()
		t.Fatalf("failed to initialize sqlite repository: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return repo, cleanup
}

func TestSQLiteRepository_CreateAndGetByID(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	p := &Pipeline{
		ID:   "p-1",
		Name: "Data Extraction Pipeline",
		Nodes: []Node{
			{
				ID:     "node-1",
				Type:   "extract",
				Config: json.RawMessage(`{"source":"db"}`),
			},
		},
		Edges: []Edge{
			{
				From: "node-1",
				To:   "node-2",
			},
		},
	}

	// Create
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("unexpected error creating pipeline: %v", err)
	}

	// Create duplicate ID should return ErrAlreadyExists
	if err := repo.Create(ctx, p); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}

	// GetByID existing
	got, err := repo.GetByID(ctx, "p-1")
	if err != nil {
		t.Fatalf("unexpected error getting pipeline: %v", err)
	}

	if got.ID != p.ID {
		t.Errorf("expected ID %q, got %q", p.ID, got.ID)
	}
	if got.Name != p.Name {
		t.Errorf("expected Name %q, got %q", p.Name, got.Name)
	}
	if !reflect.DeepEqual(got.Nodes, p.Nodes) {
		t.Errorf("expected Nodes %+v, got %+v", p.Nodes, got.Nodes)
	}
	if !reflect.DeepEqual(got.Edges, p.Edges) {
		t.Errorf("expected Edges %+v, got %+v", p.Edges, got.Edges)
	}

	// GetByID non-existing
	_, err = repo.GetByID(ctx, "non-existing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSQLiteRepository_Create_NilPipeline(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	err := repo.Create(ctx, nil)
	if err == nil {
		t.Fatal("expected error when pipeline is nil, got nil")
	}
}

func TestSQLiteRepository_List(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// List when empty
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error listing empty pipelines: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 pipelines, got %d", len(list))
	}

	// Insert multiple
	p1 := &Pipeline{ID: "p-1", Name: "Pipeline 1", Nodes: []Node{}, Edges: []Edge{}}
	p2 := &Pipeline{ID: "p-2", Name: "Pipeline 2", Nodes: []Node{}, Edges: []Edge{}}

	if err := repo.Create(ctx, p1); err != nil {
		t.Fatalf("failed to create p1: %v", err)
	}
	if err := repo.Create(ctx, p2); err != nil {
		t.Fatalf("failed to create p2: %v", err)
	}

	list, err = repo.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error listing pipelines: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 pipelines, got %d", len(list))
	}
	if list[0].ID != "p-1" || list[1].ID != "p-2" {
		t.Errorf("unexpected list order or contents: %+v", list)
	}
}

func TestSQLiteRepository_Update(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	p := &Pipeline{
		ID:    "p-update",
		Name:  "Initial Name",
		Nodes: []Node{},
		Edges: []Edge{},
	}

	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	// Update existing
	p.Name = "Updated Name"
	p.Nodes = []Node{
		{ID: "n1", Type: "transform", Config: json.RawMessage(`{}`)},
	}
	if err := repo.Update(ctx, p); err != nil {
		t.Fatalf("unexpected error updating pipeline: %v", err)
	}

	updated, err := repo.GetByID(ctx, "p-update")
	if err != nil {
		t.Fatalf("unexpected error getting updated pipeline: %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("expected updated name, got %q", updated.Name)
	}
	if len(updated.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(updated.Nodes))
	}

	// Update non-existing
	nonExisting := &Pipeline{ID: "unknown", Name: "Unknown"}
	if err := repo.Update(ctx, nonExisting); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound when updating unknown pipeline, got %v", err)
	}

	// Update nil
	if err := repo.Update(ctx, nil); err == nil {
		t.Fatal("expected error when updating nil pipeline")
	}
}

func TestSQLiteRepository_Delete(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	p := &Pipeline{ID: "p-delete", Name: "To Delete", Nodes: []Node{}, Edges: []Edge{}}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	// Delete existing
	if err := repo.Delete(ctx, "p-delete"); err != nil {
		t.Fatalf("unexpected error deleting pipeline: %v", err)
	}

	// Verify it's deleted
	_, err := repo.GetByID(ctx, "p-delete")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after deletion, got %v", err)
	}

	// Delete non-existing
	if err := repo.Delete(ctx, "non-existing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound when deleting non-existing pipeline, got %v", err)
	}
}
