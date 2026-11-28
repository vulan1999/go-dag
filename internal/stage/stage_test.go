package stage

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

type customStage struct{}

func (c *customStage) Type() string { return "custom" }
func (c *customStage) Execute(_ context.Context, inputs []Record) ([]Record, error) {
	return inputs, nil
}

func TestRegistry_DefaultAndCustom(t *testing.T) {
	reg := NewDefaultRegistry()

	wantTypes := []string{"db_sink", "db_source", "filter", "sink", "source", "transform"}
	if got := reg.Types(); !reflect.DeepEqual(got, wantTypes) {
		t.Errorf("expected default types %v, got %v", wantTypes, got)
	}

	// Unknown stage type
	if _, err := reg.Create("does_not_exist", nil); !errors.Is(err, ErrUnknownStageType) {
		t.Fatalf("expected ErrUnknownStageType, got %v", err)
	}

	// Register custom stage
	reg.Register("custom", func(_ json.RawMessage) (Stage, error) {
		return &customStage{}, nil
	})
	s, err := reg.Create("custom", nil)
	if err != nil {
		t.Fatalf("unexpected error creating custom stage: %v", err)
	}
	if s.Type() != "custom" {
		t.Errorf("expected type custom, got %q", s.Type())
	}

	// Invalid config returns ErrInvalidConfig
	if _, err := reg.Create("source", json.RawMessage(`{invalid`)); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}
}
