package components

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestPipelineRow_Render(t *testing.T) {
	view := PipelineView{
		ID:        "abc-123",
		Name:      "Sample Pipeline",
		NodesJSON: "[]",
		EdgesJSON: "[]",
		NodeCount: 2,
		EdgeCount: 1,
	}

	var buf bytes.Buffer
	if err := PipelineRow(view).Render(context.Background(), &buf); err != nil {
		t.Fatalf("failed to render PipelineRow: %v", err)
	}

	html := buf.String()
	for _, want := range []string{"pipeline-abc-123", "Sample Pipeline", `hx-get="/pipelines/abc-123"`, `hx-delete="/pipelines/abc-123"`} {
		if !strings.Contains(html, want) {
			t.Errorf("expected rendered HTML to contain %q", want)
		}
	}
}

func TestPipelineEditRow_Render(t *testing.T) {
	view := PipelineView{
		ID:        "abc-123",
		Name:      "Sample Pipeline",
		NodesJSON: "[]",
		EdgesJSON: "[]",
	}

	var buf bytes.Buffer
	if err := PipelineEditRow(view, "Invalid JSON").Render(context.Background(), &buf); err != nil {
		t.Fatalf("failed to render PipelineEditRow: %v", err)
	}

	html := buf.String()
	for _, want := range []string{`hx-put="/pipelines/abc-123"`, "Sample Pipeline", "Invalid JSON"} {
		if !strings.Contains(html, want) {
			t.Errorf("expected rendered HTML to contain %q", want)
		}
	}
}
