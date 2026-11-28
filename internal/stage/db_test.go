package stage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
)

func TestDBSourceAndSink_ETL(t *testing.T) {
	tmpDir := t.TempDir()
	dbAPath := filepath.Join(tmpDir, "db_a.sqlite")
	dbBPath := filepath.Join(tmpDir, "db_b.sqlite")

	// Setup DB A with source data
	dbA, err := sql.Open("sqlite3", dbAPath)
	if err != nil {
		t.Fatalf("failed to open db_a: %v", err)
	}
	defer dbA.Close()

	if _, err := dbA.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, active INTEGER);
		INSERT INTO users (id, name, active) VALUES (1, 'Alice', 1), (2, 'Bob', 0);`); err != nil {
		t.Fatalf("failed to seed db_a: %v", err)
	}

	// Setup DB B with target table
	dbB, err := sql.Open("sqlite3", dbBPath)
	if err != nil {
		t.Fatalf("failed to open db_b: %v", err)
	}
	defer dbB.Close()

	if _, err := dbB.Exec(`CREATE TABLE target_users (id INTEGER PRIMARY KEY, full_name TEXT);`); err != nil {
		t.Fatalf("failed to create target_users in db_b: %v", err)
	}

	// 1. Extract from DB A
	srcCfg := fmt.Sprintf(`{"dsn": %q, "query": "SELECT id, name, active FROM users"}`, dbAPath)
	src, err := NewDBSourceStage(json.RawMessage(srcCfg))
	if err != nil {
		t.Fatalf("failed to create db_source: %v", err)
	}

	rows, err := src.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("db_source execute failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows from db_a, got %d", len(rows))
	}

	// 2. Filter active == 1
	filter, err := NewFilterStage(json.RawMessage(`{"field":"active","op":"eq","value":1}`))
	if err != nil {
		t.Fatalf("failed to create filter: %v", err)
	}
	filtered, err := filter.Execute(context.Background(), rows)
	if err != nil {
		t.Fatalf("filter execute failed: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 active row, got %d", len(filtered))
	}

	// 3. Transform name -> full_name
	transform, err := NewTransformStage(json.RawMessage(`{"rename":{"name":"full_name"}}`))
	if err != nil {
		t.Fatalf("failed to create transform: %v", err)
	}
	transformed, err := transform.Execute(context.Background(), filtered)
	if err != nil {
		t.Fatalf("transform execute failed: %v", err)
	}

	// 4. Load into DB B
	sinkCfg := fmt.Sprintf(`{"dsn": %q, "table": "target_users", "columns": ["id", "full_name"]}`, dbBPath)
	sink, err := NewDBSinkStage(json.RawMessage(sinkCfg))
	if err != nil {
		t.Fatalf("failed to create db_sink: %v", err)
	}
	if _, err := sink.Execute(context.Background(), transformed); err != nil {
		t.Fatalf("db_sink execute failed: %v", err)
	}

	// Verify DB B contents
	var gotID int
	var gotName string
	if err := dbB.QueryRow(`SELECT id, full_name FROM target_users`).Scan(&gotID, &gotName); err != nil {
		t.Fatalf("failed to query db_b: %v", err)
	}
	if gotID != 1 || gotName != "Alice" {
		t.Errorf("expected (1, Alice) in db_b, got (%d, %q)", gotID, gotName)
	}
}

func TestDBStages_ConfigValidation(t *testing.T) {
	if _, err := NewDBSourceStage(json.RawMessage(`{}`)); !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig for missing dsn, got %v", err)
	}
	if _, err := NewDBSourceStage(json.RawMessage(`{"dsn":"a.db"}`)); !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig for missing query, got %v", err)
	}
	if _, err := NewDBSinkStage(json.RawMessage(`{"dsn":"b.db","table":"bad-name!","columns":["id"]}`)); !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig for invalid table name, got %v", err)
	}
	if _, err := NewDBSinkStage(json.RawMessage(`{"dsn":"b.db","table":"users","columns":["bad col"]}`)); !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig for invalid column name, got %v", err)
	}
}
