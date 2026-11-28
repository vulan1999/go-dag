package components

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestStageConfigFields_Render(t *testing.T) {
	t.Run("db_source", func(t *testing.T) {
		var buf bytes.Buffer
		if err := StageConfigFields("db_source", nil).Render(context.Background(), &buf); err != nil {
			t.Fatalf("failed to render db_source fields: %v", err)
		}
		html := buf.String()
		for _, want := range []string{`name="driver"`, `name="dsn"`, `name="query"`, `name="columns"`} {
			if !strings.Contains(html, want) {
				t.Errorf("expected HTML to contain %q", want)
			}
		}
	})

	t.Run("transform", func(t *testing.T) {
		var buf bytes.Buffer
		if err := StageConfigFields("transform", []string{"id", "name"}).Render(context.Background(), &buf); err != nil {
			t.Fatalf("failed to render transform fields: %v", err)
		}
		html := buf.String()
		for _, want := range []string{`name="rename_from[]"`, `name="rename_to[]"`, `value="id"`, `value="name"`, "+ Add Mapping", "+ Add Static Field"} {
			if !strings.Contains(html, want) {
				t.Errorf("expected HTML to contain %q", want)
			}
		}
	})

	t.Run("db_sink with transformed columns", func(t *testing.T) {
		var buf bytes.Buffer
		if err := StageConfigFields("db_sink", []string{"id", "full_name"}).Render(context.Background(), &buf); err != nil {
			t.Fatalf("failed to render db_sink fields: %v", err)
		}
		html := buf.String()
		for _, want := range []string{`name="table"`, `name="columns"`, "Transformed Columns:", "id, full_name"} {
			if !strings.Contains(html, want) {
				t.Errorf("expected HTML to contain %q", want)
			}
		}
	})

	t.Run("NodeEditCard prefilled", func(t *testing.T) {
		var buf bytes.Buffer
		vals := StageConfigValues{
			Driver:  "sqlite3",
			DSN:     "db_a.sqlite",
			Query:   "SELECT id, name FROM users",
			Columns: "id, name",
		}
		if err := NodeEditCard("p-1", "src_a", "db_source", vals, nil, "").Render(context.Background(), &buf); err != nil {
			t.Fatalf("failed to render NodeEditCard: %v", err)
		}
		html := buf.String()
		for _, want := range []string{`hx-put="/pipelines/p-1/nodes/src_a"`, "db_a.sqlite", "SELECT id, name FROM users", "id, name"} {
			if !strings.Contains(html, want) {
				t.Errorf("expected HTML to contain %q", want)
			}
		}
	})
}
