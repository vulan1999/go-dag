package stage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
)

// SourceConfig defines static records to emit.
type SourceConfig struct {
	Records []Record `json:"records"`
}

type SourceStage struct {
	records []Record
}

func NewSourceStage(raw json.RawMessage) (Stage, error) {
	var cfg SourceConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	return &SourceStage{records: cfg.Records}, nil
}

func (s *SourceStage) Type() string { return "source" }

func (s *SourceStage) Execute(ctx context.Context, _ []Record) ([]Record, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]Record, 0, len(s.records))
	for _, rec := range s.records {
		out = append(out, maps.Clone(rec))
	}
	return out, nil
}

// FilterConfig defines a predicate over a record field.
type FilterConfig struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value"`
}

type FilterStage struct {
	cfg FilterConfig
}

func NewFilterStage(raw json.RawMessage) (Stage, error) {
	var cfg FilterConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	if cfg.Field == "" {
		return nil, fmt.Errorf("%w: field is required", ErrInvalidConfig)
	}
	if cfg.Op == "" {
		cfg.Op = "eq"
	}
	switch cfg.Op {
	case "eq", "neq", "exists":
	default:
		return nil, fmt.Errorf("%w: unsupported op %q", ErrInvalidConfig, cfg.Op)
	}
	return &FilterStage{cfg: cfg}, nil
}

func (s *FilterStage) Type() string { return "filter" }

func (s *FilterStage) Execute(ctx context.Context, inputs []Record) ([]Record, error) {
	out := make([]Record, 0, len(inputs))
	for _, rec := range inputs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		val, exists := rec[s.cfg.Field]
		match := false
		switch s.cfg.Op {
		case "exists":
			match = exists
		case "eq":
			match = exists && fmt.Sprint(val) == fmt.Sprint(s.cfg.Value)
		case "neq":
			match = !exists || fmt.Sprint(val) != fmt.Sprint(s.cfg.Value)
		}
		if match {
			out = append(out, maps.Clone(rec))
		}
	}
	return out, nil
}

// TransformConfig defines field renames and static field assignments.
type TransformConfig struct {
	Rename map[string]string `json:"rename"`
	Set    map[string]any    `json:"set"`
}

type TransformStage struct {
	cfg TransformConfig
}

func NewTransformStage(raw json.RawMessage) (Stage, error) {
	var cfg TransformConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	return &TransformStage{cfg: cfg}, nil
}

func (s *TransformStage) Type() string { return "transform" }

func (s *TransformStage) Execute(ctx context.Context, inputs []Record) ([]Record, error) {
	out := make([]Record, 0, len(inputs))
	for _, rec := range inputs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		cloned := maps.Clone(rec)
		if cloned == nil {
			cloned = Record{}
		}
		for oldKey, newKey := range s.cfg.Rename {
			if val, ok := cloned[oldKey]; ok {
				delete(cloned, oldKey)
				cloned[newKey] = val
			}
		}
		for k, v := range s.cfg.Set {
			cloned[k] = v
		}
		out = append(out, cloned)
	}
	return out, nil
}

// SinkStage writes incoming records as JSON lines and passes them through.
type SinkStage struct {
	writer io.Writer
}

func NewSinkStage(raw json.RawMessage) (Stage, error) {
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	return &SinkStage{writer: os.Stdout}, nil
}

func (s *SinkStage) Type() string { return "sink" }

func (s *SinkStage) Execute(ctx context.Context, inputs []Record) ([]Record, error) {
	out := make([]Record, 0, len(inputs))
	for _, rec := range inputs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		line, err := json.Marshal(rec)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal record: %w", err)
		}
		if _, err := fmt.Fprintln(s.writer, string(line)); err != nil {
			return nil, fmt.Errorf("failed to write record: %w", err)
		}
		out = append(out, maps.Clone(rec))
	}
	return out, nil
}
