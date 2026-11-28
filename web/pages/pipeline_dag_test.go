package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/vulan1999/go-dag/web/components"
)

func TestPipelineDAG_Render(t *testing.T) {
	view := components.DAGView{
		PipelineID:   "p-1",
		PipelineName: "Daily ETL",
		Valid:        true,
	}

	var buf bytes.Buffer
	if err := PipelineDAG(view).Render(context.Background(), &buf); err != nil {
		t.Fatalf("failed to render PipelineDAG: %v", err)
	}

	html := buf.String()
	for _, want := range []string{"Daily ETL", "ID: p-1", `/static/js/dag.js`, `/static/js/htmx.min.js`} {
		if !strings.Contains(html, want) {
			t.Errorf("expected rendered HTML to contain %q", want)
		}
	}
}
