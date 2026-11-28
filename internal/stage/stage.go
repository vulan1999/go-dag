package stage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

var (
	ErrUnknownStageType = errors.New("unknown stage type")
	ErrInvalidConfig    = errors.New("invalid stage config")
)

// Record represents a single data row flowing between pipeline stages.
type Record map[string]any

// Stage defines a single executable unit in a pipeline DAG.
type Stage interface {
	Type() string
	Execute(ctx context.Context, inputs []Record) ([]Record, error)
}

// Constructor instantiates a Stage from raw JSON configuration.
type Constructor func(config json.RawMessage) (Stage, error)

// Registry maps stage type names to their constructors.
type Registry struct {
	constructors map[string]Constructor
}

// NewRegistry creates an empty stage registry.
func NewRegistry() *Registry {
	return &Registry{
		constructors: make(map[string]Constructor),
	}
}

// NewDefaultRegistry creates a registry pre-populated with all built-in stages.
func NewDefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register("source", NewSourceStage)
	r.Register("filter", NewFilterStage)
	r.Register("transform", NewTransformStage)
	r.Register("sink", NewSinkStage)
	r.Register("db_source", NewDBSourceStage)
	r.Register("db_sink", NewDBSinkStage)
	return r
}

// Register binds a stage type name to a constructor.
func (r *Registry) Register(stageType string, fn Constructor) {
	if strings.TrimSpace(stageType) == "" || fn == nil {
		return
	}
	r.constructors[stageType] = fn
}

// Create instantiates a stage by type name and JSON config.
func (r *Registry) Create(stageType string, config json.RawMessage) (Stage, error) {
	fn, ok := r.constructors[stageType]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownStageType, stageType)
	}
	if len(config) == 0 {
		config = json.RawMessage("{}")
	}
	return fn(config)
}

// Types returns a sorted list of registered stage type names.
func (r *Registry) Types() []string {
	types := make([]string, 0, len(r.constructors))
	for t := range r.constructors {
		types = append(types, t)
	}
	slices.Sort(types)
	return types
}
