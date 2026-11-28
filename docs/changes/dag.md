# DAG Implementation

## Overview
Implemented the core Directed Acyclic Graph (DAG) structure and topological scheduling helpers in `internal/dag`.

## Changes

### 1. DAG Structure & Methods (`internal/dag/dag.go`)
- Defined sentinel errors:
  - `ErrInvalidNodeID`
  - `ErrDuplicateNode`
  - `ErrNodeNotFound`
  - `ErrDuplicateEdge`
  - `ErrCycleDetected`
- Implemented `DAG` struct with:
  - `New() *DAG`
  - `AddNode(id string) error`
  - `AddEdge(from, to string) error`
  - `Validate() error`
  - `TopologicalSort() ([]string, error)`
  - `Levels() ([][]string, error)` (Kahn's algorithm grouped by execution wave for concurrent execution)
  - `Nodes() []string`, `Parents(id string) []string`, `Children(id string) []string`

### 2. Unit Tests (`internal/dag/dag_test.go`)
- `TestDAG_AddNode`: verifies valid insertion, duplicate rejection, and empty/whitespace ID validation.
- `TestDAG_AddEdge`: verifies edge creation, duplicate edge rejection, self-loop cycle detection, missing node errors, and `Parents`/`Children` helpers.
- `TestDAG_TopologicalSortAndLevels`: verifies empty graphs, linear chains, diamond graphs, disconnected components, and cycle detection.
