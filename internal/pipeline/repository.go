package pipeline

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mattn/go-sqlite3"
)

var (
	ErrNotFound      = errors.New("pipeline not found")
	ErrAlreadyExists = errors.New("pipeline already exists")
)

type Repository interface {
	Create(ctx context.Context, p *Pipeline) error
	GetByID(ctx context.Context, id string) (*Pipeline, error)
	List(ctx context.Context) ([]*Pipeline, error)
	Update(ctx context.Context, p *Pipeline) error
	Delete(ctx context.Context, id string) error
}

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) (*SQLiteRepository, error) {
	repo := &SQLiteRepository{db: db}
	if err := repo.initSchema(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize pipeline schema: %w", err)
	}
	return repo, nil
}

func (r *SQLiteRepository) initSchema(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS pipelines (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		nodes TEXT NOT NULL,
		edges TEXT NOT NULL
	);`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

func (r *SQLiteRepository) Create(ctx context.Context, p *Pipeline) error {
	if p == nil {
		return errors.New("pipeline cannot be nil")
	}

	nodes := p.Nodes
	if nodes == nil {
		nodes = []Node{}
	}
	edges := p.Edges
	if edges == nil {
		edges = []Edge{}
	}

	nodesJSON, err := json.Marshal(nodes)
	if err != nil {
		return fmt.Errorf("failed to marshal nodes: %w", err)
	}

	edgesJSON, err := json.Marshal(edges)
	if err != nil {
		return fmt.Errorf("failed to marshal edges: %w", err)
	}

	query := `INSERT INTO pipelines (id, name, nodes, edges) VALUES (?, ?, ?, ?)`
	_, err = r.db.ExecContext(ctx, query, p.ID, p.Name, string(nodesJSON), string(edgesJSON))
	if err != nil {
		var sqliteErr sqlite3.Error
		if (errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey) ||
			strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrAlreadyExists
		}
		return fmt.Errorf("failed to insert pipeline: %w", err)
	}

	return nil
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (*Pipeline, error) {
	query := `SELECT id, name, nodes, edges FROM pipelines WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		p         Pipeline
		nodesJSON string
		edgesJSON string
	)

	if err := row.Scan(&p.ID, &p.Name, &nodesJSON, &edgesJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get pipeline by id: %w", err)
	}

	if err := json.Unmarshal([]byte(nodesJSON), &p.Nodes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal nodes: %w", err)
	}

	if err := json.Unmarshal([]byte(edgesJSON), &p.Edges); err != nil {
		return nil, fmt.Errorf("failed to unmarshal edges: %w", err)
	}

	return &p, nil
}

func (r *SQLiteRepository) List(ctx context.Context) ([]*Pipeline, error) {
	query := `SELECT id, name, nodes, edges FROM pipelines ORDER BY id`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}
	defer rows.Close()

	pipelines := make([]*Pipeline, 0)
	for rows.Next() {
		var (
			p         Pipeline
			nodesJSON string
			edgesJSON string
		)
		if err := rows.Scan(&p.ID, &p.Name, &nodesJSON, &edgesJSON); err != nil {
			return nil, fmt.Errorf("failed to scan pipeline row: %w", err)
		}
		if err := json.Unmarshal([]byte(nodesJSON), &p.Nodes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal nodes: %w", err)
		}
		if err := json.Unmarshal([]byte(edgesJSON), &p.Edges); err != nil {
			return nil, fmt.Errorf("failed to unmarshal edges: %w", err)
		}
		pipelines = append(pipelines, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate pipeline rows: %w", err)
	}

	return pipelines, nil
}

func (r *SQLiteRepository) Update(ctx context.Context, p *Pipeline) error {
	if p == nil {
		return errors.New("pipeline cannot be nil")
	}

	nodes := p.Nodes
	if nodes == nil {
		nodes = []Node{}
	}
	edges := p.Edges
	if edges == nil {
		edges = []Edge{}
	}

	nodesJSON, err := json.Marshal(nodes)
	if err != nil {
		return fmt.Errorf("failed to marshal nodes: %w", err)
	}

	edgesJSON, err := json.Marshal(edges)
	if err != nil {
		return fmt.Errorf("failed to marshal edges: %w", err)
	}

	query := `UPDATE pipelines SET name = ?, nodes = ?, edges = ? WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, p.Name, string(nodesJSON), string(edgesJSON), p.ID)
	if err != nil {
		return fmt.Errorf("failed to update pipeline: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *SQLiteRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM pipelines WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete pipeline: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
