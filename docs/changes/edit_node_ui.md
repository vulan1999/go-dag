# Node Property Editing UI

## Overview
Added click-to-edit support for existing nodes on the DAG canvas using HTMX and pre-filled stage configuration forms.

## Changes

### 1. Frontend Templates & Styles
- **`web/components/dag_canvas.templ`**:
  - Made `.dag-node` cards clickable (`hx-get="/pipelines/{id}/nodes/{nodeId}"` targeting `#node-inspector`).
  - Stopped click propagation on the `Remove` button (`hx-on:click="event.stopPropagation()"`).
  - Added `<div id="node-inspector"></div>` below the workflow canvas.
- **`web/components/stage_config.templ`**:
  - Added `StageConfigValues` and `StageConfigFieldsWithValues` to share field templates between Add and Edit forms.
  - Added `NodeEditCard` with `Stage Type` selector, pre-filled config inputs, **Save Changes** (`hx-put`), and **Cancel**.
- **`web/static/css/styles.css`**:
  - Added pointer cursor and hover highlight to `.dag-node`.

### 2. Backend Handlers & Routes
- **`internal/pipeline/dag_handler.go`**:
  - Added `GetNode` (`GET /pipelines/{id}/nodes/{nodeId}`) to render `NodeEditCard` pre-filled with the node's current properties.
  - Added `UpdateNode` (`PUT /pipelines/{id}/nodes/{nodeId}`) to validate and persist updated stage type and config, then refresh `#dag-workspace`.
- **`internal/server/server.go`**:
  - Registered `GET /pipelines/{id}/nodes/{nodeId}` and `PUT /pipelines/{id}/nodes/{nodeId}`.

### 3. Unit Tests
- `web/components/stage_config_test.go`: Added `NodeEditCard prefilled` test.
- `internal/pipeline/dag_handler_test.go`: Added `GetNode` and `UpdateNode` assertions to `TestHandler_StageConfigFieldsAndTransformedColumns`.
