package stage

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseSelectColumns(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{
			name:  "simple select",
			query: "SELECT id, name, email FROM users",
			want:  []string{"id", "name", "email"},
		},
		{
			name:  "distinct and table prefixes",
			query: "SELECT DISTINCT u.id, u.name FROM users u",
			want:  []string{"id", "name"},
		},
		{
			name:  "aliases and functions with commas",
			query: `SELECT id, COALESCE(first_name, 'N/A') AS name, COUNT(*) total FROM users`,
			want:  []string{"id", "name", "total"},
		},
		{
			name:  "scalar subquery in select",
			query: "SELECT u.id, (SELECT COUNT(*) FROM orders o WHERE o.user_id = u.id) AS order_count FROM users u",
			want:  []string{"id", "order_count"},
		},
		{
			name:  "nested subquery in from",
			query: "SELECT id, total_amount FROM (SELECT user_id AS id, SUM(amount) AS total_amount FROM orders GROUP BY user_id) sub",
			want:  []string{"id", "total_amount"},
		},
		{
			name:  "select star from nested subquery",
			query: "SELECT * FROM (SELECT id, full_name FROM users) sub",
			want:  []string{"id", "full_name"},
		},
		{
			name:  "CTE query",
			query: "WITH active_users AS (SELECT id, name FROM users WHERE active = 1) SELECT id, name AS user_name FROM active_users;",
			want:  []string{"id", "user_name"},
		},
		{
			name:  "select star on table",
			query: "SELECT * FROM users",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSelectColumns(tt.query)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseSelectColumns() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntrospectQueryColumns(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.sqlite")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE users (id INTEGER, name TEXT, email TEXT);`); err != nil {
		db.Close()
		t.Fatalf("failed to create table: %v", err)
	}
	db.Close()

	cols, err := IntrospectQueryColumns(context.Background(), "sqlite3", dbPath, "SELECT * FROM users")
	if err != nil {
		t.Fatalf("unexpected introspection error: %v", err)
	}
	want := []string{"id", "name", "email"}
	if !reflect.DeepEqual(cols, want) {
		t.Errorf("got %v, want %v", cols, want)
	}
}
