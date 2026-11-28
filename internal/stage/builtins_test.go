package stage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestSourceStage(t *testing.T) {
	s, err := NewSourceStage(json.RawMessage(`{"records":[{"id":1,"name":"alice"}]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Type() != "source" {
		t.Errorf("expected type source, got %q", s.Type())
	}

	out, err := s.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}
	if len(out) != 1 || out[0]["name"] != "alice" {
		t.Errorf("unexpected output: %+v", out)
	}

	// Cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Execute(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestFilterStage(t *testing.T) {
	if _, err := NewFilterStage(json.RawMessage(`{}`)); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig when field is missing, got %v", err)
	}
	if _, err := NewFilterStage(json.RawMessage(`{"field":"status","op":"bad"}`)); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig for unsupported op, got %v", err)
	}

	inputs := []Record{
		{"id": 1, "status": "active"},
		{"id": 2, "status": "inactive"},
		{"id": 3},
	}

	t.Run("eq", func(t *testing.T) {
		s, err := NewFilterStage(json.RawMessage(`{"field":"status","op":"eq","value":"active"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out, err := s.Execute(context.Background(), inputs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(out) != 1 || out[0]["id"] != 1 {
			t.Errorf("expected 1 matching record, got %+v", out)
		}
	})

	t.Run("neq", func(t *testing.T) {
		s, err := NewFilterStage(json.RawMessage(`{"field":"status","op":"neq","value":"active"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out, err := s.Execute(context.Background(), inputs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(out) != 2 {
			t.Errorf("expected 2 records, got %+v", out)
		}
	})

	t.Run("exists", func(t *testing.T) {
		s, err := NewFilterStage(json.RawMessage(`{"field":"status","op":"exists"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out, err := s.Execute(context.Background(), inputs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(out) != 2 {
			t.Errorf("expected 2 records, got %+v", out)
		}
	})
}

func TestTransformStage(t *testing.T) {
	s, err := NewTransformStage(json.RawMessage(`{"rename":{"full_name":"name"},"set":{"env":"prod"}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Type() != "transform" {
		t.Errorf("expected type transform, got %q", s.Type())
	}

	out, err := s.Execute(context.Background(), []Record{{"full_name": "alice"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, oldExists := out[0]["full_name"]; oldExists {
		t.Errorf("expected full_name to be renamed")
	}
	if out[0]["name"] != "alice" || out[0]["env"] != "prod" {
		t.Errorf("unexpected transformed record: %+v", out[0])
	}
}

func TestSinkStage(t *testing.T) {
	var buf bytes.Buffer
	s := &SinkStage{writer: &buf}
	if s.Type() != "sink" {
		t.Errorf("expected type sink, got %q", s.Type())
	}

	out, err := s.Execute(context.Background(), []Record{{"id": 1}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Errorf("expected passthrough of 1 record, got %d", len(out))
	}
	if !strings.Contains(buf.String(), `"id":1`) {
		t.Errorf("expected JSON line output, got %q", buf.String())
	}
}
