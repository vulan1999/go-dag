package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/vulan1999/go-dag/web/components"
)

func TestPipelinesPage_Render(t *testing.T) {
	items := []components.PipelineView{
		{
			ID:        "p-1",
			Name:      "Test Flow",
			NodesJSON: "[]",
			EdgesJSON: "[]",
			NodeCount: 1,
			EdgeCount: 0,
		},
	}

	var buf bytes.Buffer
	if err := Pipelines(items).Render(context.Background(), &buf); err != nil {
		t.Fatalf("failed to render Pipelines page: %v", err)
	}

	html := buf.String()
	for _, want := range []string{"Pipelines", `hx-post="/pipelines"`, "Test Flow", "/static/styles.css"} {
		if !strings.Contains(html, want) {
			t.Errorf("expected rendered page to contain %q", want)
		}
	}
}
