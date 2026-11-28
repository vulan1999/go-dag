# DAG Frontend Implementation

## Overview
Implemented an interactive, DataStage-inspired DAG designer and visualizer using Go, Templ, HTMX, and a lightweight SVG connector script.

## Changes

### 1. Frontend Templates & Assets
- **`web/layouts/base.templ`**: Updated HTMX script path to `/static/js/htmx.min.js` so nested routes (`/pipelines/{id}/dag`) load HTMX properly.
- **`web/components/pipeline.templ`**: Added a `DAG` link in `PipelineRow` navigating to `/pipelines/{id}/dag`.
- **`web/components/dag_canvas.templ`**: Added `DAGWorkspace`, `DAGView`, `DAGNodeView`, and `DAGEdgeView` to render:
  - Validation status badge (`Valid DAG` vs `Invalid DAG`) and form error alerts.
  - Wave-based left-to-right stage columns computed from `internal/dag.Levels()`.
  - Interactive HTMX forms for adding stages, connecting stages (`from -> to`), and removing stages/edges.
- **`web/pages/pipeline_dag.templ`**: Added the `PipelineDAG` page wrapping `DAGWorkspace` and loading `/static/js/dag.js`.
- **`web/static/js/dag.js`**: Draws Bezier SVG arrows between stage cards on load, resize, and `htmx:afterSwap`.
- **`web/static/css/styles.css`**: Added styling for the DAG canvas, wave columns, stage cards, badges, and controls.

### 2. Backend Handlers & Routes
- **`internal/pipeline/dag_handler.go`**: Added `GetDAG`, `AddNode`, `DeleteNode`, `AddEdge`, and `DeleteEdge` handlers, validating graph mutations with `internal/dag` before persisting.
- **`internal/server/server.go`**: Registered `/pipelines/{id}/dag`, `/pipelines/{id}/nodes`, `/pipelines/{id}/nodes/{nodeId}`, and `/pipelines/{id}/edges` routes.

### 3. Unit Tests
- `internal/pipeline/dag_handler_test.go`: Tests `GetDAG`, `AddNode`, `DeleteNode`, `AddEdge`, `DeleteEdge`, and cycle rejection.
- `web/components/dag_canvas_test.go`: Tests rendering `DAGWorkspace`.
- `web/pages/pipeline_dag_test.go`: Tests rendering `PipelineDAG`.
