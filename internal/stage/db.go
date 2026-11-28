package stage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var validIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// DBSourceConfig configures reading records from a SQL database.
type DBSourceConfig struct {
	Driver  string   `json:"driver"`
	DSN     string   `json:"dsn"`
	Query   string   `json:"query"`
	Columns []string `json:"columns,omitempty"`
}

type DBSourceStage struct {
	cfg DBSourceConfig
}

func NewDBSourceStage(raw json.RawMessage) (Stage, error) {
	var cfg DBSourceConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	if strings.TrimSpace(cfg.Driver) == "" {
		cfg.Driver = "sqlite3"
	}
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, fmt.Errorf("%w: dsn is required", ErrInvalidConfig)
	}
	if strings.TrimSpace(cfg.Query) == "" {
		return nil, fmt.Errorf("%w: query is required", ErrInvalidConfig)
	}
	return &DBSourceStage{cfg: cfg}, nil
}

func (s *DBSourceStage) Type() string { return "db_source" }

func (s *DBSourceStage) Execute(ctx context.Context, _ []Record) ([]Record, error) {
	db, err := sql.Open(s.cfg.Driver, s.cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, s.cfg.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to read columns: %w", err)
	}

	out := make([]Record, 0)
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		rec := make(Record, len(cols))
		for i, col := range cols {
			if b, ok := values[i].([]byte); ok {
				rec[col] = string(b)
			} else {
				rec[col] = values[i]
			}
		}
		out = append(out, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return out, nil
}

// DBSinkConfig configures writing records into a SQL table.
type DBSinkConfig struct {
	Driver  string   `json:"driver"`
	DSN     string   `json:"dsn"`
	Table   string   `json:"table"`
	Columns []string `json:"columns"`
}

type DBSinkStage struct {
	cfg DBSinkConfig
}

func NewDBSinkStage(raw json.RawMessage) (Stage, error) {
	var cfg DBSinkConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	if strings.TrimSpace(cfg.Driver) == "" {
		cfg.Driver = "sqlite3"
	}
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, fmt.Errorf("%w: dsn is required", ErrInvalidConfig)
	}
	if !validIdentifier.MatchString(cfg.Table) {
		return nil, fmt.Errorf("%w: invalid table name %q", ErrInvalidConfig, cfg.Table)
	}
	if len(cfg.Columns) == 0 {
		return nil, fmt.Errorf("%w: columns are required", ErrInvalidConfig)
	}
	for _, col := range cfg.Columns {
		if !validIdentifier.MatchString(col) {
			return nil, fmt.Errorf("%w: invalid column name %q", ErrInvalidConfig, col)
		}
	}
	return &DBSinkStage{cfg: cfg}, nil
}

func (s *DBSinkStage) Type() string { return "db_sink" }

func (s *DBSinkStage) Execute(ctx context.Context, inputs []Record) ([]Record, error) {
	if len(inputs) == 0 {
		return []Record{}, nil
	}

	db, err := sql.Open(s.cfg.Driver, s.cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	placeholders := make([]string, len(s.cfg.Columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		s.cfg.Table,
		strings.Join(s.cfg.Columns, ", "),
		strings.Join(placeholders, ", "),
	)

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	out := make([]Record, 0, len(inputs))
	for _, rec := range inputs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		args := make([]any, len(s.cfg.Columns))
		for i, col := range s.cfg.Columns {
			args[i] = rec[col]
		}
		if _, err := stmt.ExecContext(ctx, args...); err != nil {
			return nil, fmt.Errorf("failed to insert record: %w", err)
		}
		out = append(out, maps.Clone(rec))
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return out, nil
}
