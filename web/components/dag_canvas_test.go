package components

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestDAGWorkspace_Render(t *testing.T) {
	view := DAGView{
		PipelineID:   "p-1",
		PipelineName: "Test DAG",
		Valid:        true,
		Nodes: []DAGNodeView{
			{ID: "extract", Type: "source", ConfigJSON: "{}"},
			{ID: "load", Type: "sink", ConfigJSON: "{}"},
		},
		Waves: [][]DAGNodeView{
			{{ID: "extract", Type: "source", ConfigJSON: "{}"}},
			{{ID: "load", Type: "sink", ConfigJSON: "{}"}},
		},
		Edges: []DAGEdgeView{
			{From: "extract", To: "load"},
		},
	}

	var buf bytes.Buffer
	if err := DAGWorkspace(view, "").Render(context.Background(), &buf); err != nil {
		t.Fatalf("failed to render DAGWorkspace: %v", err)
	}

	html := buf.String()
	for _, want := range []string{
		`id="dag-workspace"`,
		"Valid DAG",
		`data-node-id="extract"`,
		`data-node-id="load"`,
		`hx-post="/pipelines/p-1/nodes"`,
		`hx-post="/pipelines/p-1/edges"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected rendered HTML to contain %q", want)
		}
	}
}
