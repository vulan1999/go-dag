package dag

import (
	"errors"
	"reflect"
	"testing"
)

func TestDAG_AddNode(t *testing.T) {
	g := New()

	if err := g.AddNode("extract"); err != nil {
		t.Fatalf("unexpected error adding node: %v", err)
	}

	if err := g.AddNode("extract"); !errors.Is(err, ErrDuplicateNode) {
		t.Fatalf("expected ErrDuplicateNode, got %v", err)
	}

	for _, invalid := range []string{"", "   "} {
		if err := g.AddNode(invalid); !errors.Is(err, ErrInvalidNodeID) {
			t.Fatalf("expected ErrInvalidNodeID for %q, got %v", invalid, err)
		}
	}

	if got := g.Nodes(); !reflect.DeepEqual(got, []string{"extract"}) {
		t.Errorf("expected Nodes [extract], got %v", got)
	}
}

func TestDAG_AddEdge(t *testing.T) {
	g := New()
	_ = g.AddNode("a")
	_ = g.AddNode("b")

	if err := g.AddEdge("a", "b"); err != nil {
		t.Fatalf("unexpected error adding edge: %v", err)
	}

	if err := g.AddEdge("a", "b"); !errors.Is(err, ErrDuplicateEdge) {
		t.Fatalf("expected ErrDuplicateEdge, got %v", err)
	}

	if err := g.AddEdge("a", "a"); !errors.Is(err, ErrCycleDetected) {
		t.Fatalf("expected ErrCycleDetected on self-loop, got %v", err)
	}

	if err := g.AddEdge("missing", "b"); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("expected ErrNodeNotFound for missing source, got %v", err)
	}

	if err := g.AddEdge("a", "missing"); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("expected ErrNodeNotFound for missing target, got %v", err)
	}

	if got := g.Children("a"); !reflect.DeepEqual(got, []string{"b"}) {
		t.Errorf("expected Children(a) = [b], got %v", got)
	}
	if got := g.Parents("b"); !reflect.DeepEqual(got, []string{"a"}) {
		t.Errorf("expected Parents(b) = [a], got %v", got)
	}
	if got := g.Parents("a"); !reflect.DeepEqual(got, []string{}) {
		t.Errorf("expected Parents(a) = [], got %v", got)
	}
}

func TestDAG_TopologicalSortAndLevels(t *testing.T) {
	t.Run("empty graph", func(t *testing.T) {
		g := New()
		order, err := g.TopologicalSort()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(order) != 0 {
			t.Errorf("expected empty order, got %v", order)
		}
		levels, err := g.Levels()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(levels) != 0 {
			t.Errorf("expected empty levels, got %v", levels)
		}
	})

	t.Run("linear chain", func(t *testing.T) {
		g := New()
		for _, id := range []string{"a", "b", "c"} {
			_ = g.AddNode(id)
		}
		_ = g.AddEdge("a", "b")
		_ = g.AddEdge("b", "c")

		order, err := g.TopologicalSort()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantOrder := []string{"a", "b", "c"}
		if !reflect.DeepEqual(order, wantOrder) {
			t.Errorf("expected %v, got %v", wantOrder, order)
		}

		levels, err := g.Levels()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantLevels := [][]string{{"a"}, {"b"}, {"c"}}
		if !reflect.DeepEqual(levels, wantLevels) {
			t.Errorf("expected %v, got %v", wantLevels, levels)
		}
	})

	t.Run("diamond graph", func(t *testing.T) {
		g := New()
		for _, id := range []string{"extract", "transform_a", "transform_b", "load"} {
			_ = g.AddNode(id)
		}
		_ = g.AddEdge("extract", "transform_a")
		_ = g.AddEdge("extract", "transform_b")
		_ = g.AddEdge("transform_a", "load")
		_ = g.AddEdge("transform_b", "load")

		levels, err := g.Levels()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantLevels := [][]string{
			{"extract"},
			{"transform_a", "transform_b"},
			{"load"},
		}
		if !reflect.DeepEqual(levels, wantLevels) {
			t.Errorf("expected %v, got %v", wantLevels, levels)
		}

		order, err := g.TopologicalSort()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantOrder := []string{"extract", "transform_a", "transform_b", "load"}
		if !reflect.DeepEqual(order, wantOrder) {
			t.Errorf("expected %v, got %v", wantOrder, order)
		}
	})

	t.Run("disconnected components", func(t *testing.T) {
		g := New()
		for _, id := range []string{"a", "b", "c", "d"} {
			_ = g.AddNode(id)
		}
		_ = g.AddEdge("a", "b")
		_ = g.AddEdge("c", "d")

		levels, err := g.Levels()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantLevels := [][]string{{"a", "c"}, {"b", "d"}}
		if !reflect.DeepEqual(levels, wantLevels) {
			t.Errorf("expected %v, got %v", wantLevels, levels)
		}
	})

	t.Run("cycle detected", func(t *testing.T) {
		g := New()
		for _, id := range []string{"a", "b", "c"} {
			_ = g.AddNode(id)
		}
		_ = g.AddEdge("a", "b")
		_ = g.AddEdge("b", "c")
		_ = g.AddEdge("c", "a")

		if err := g.Validate(); !errors.Is(err, ErrCycleDetected) {
			t.Fatalf("expected ErrCycleDetected from Validate, got %v", err)
		}
		if _, err := g.TopologicalSort(); !errors.Is(err, ErrCycleDetected) {
			t.Fatalf("expected ErrCycleDetected from TopologicalSort, got %v", err)
		}
		if _, err := g.Levels(); !errors.Is(err, ErrCycleDetected) {
			t.Fatalf("expected ErrCycleDetected from Levels, got %v", err)
		}
	})
}
