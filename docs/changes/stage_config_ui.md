# Dynamic Stage Config UI & Transformed Column Tracking

## Overview
Replaced the raw JSON config textarea in the DAG designer with stage-type-specific inputs powered by HTMX, temporarily disabled `source`, `filter`, and `sink` in the frontend selector, and added transformed column tracking for `db_source`, `transform`, and `db_sink`.

## Changes

### 1. Stage Config UI (`web/components/stage_config.templ`, `web/components/dag_canvas.templ`)
- Limited the `Stage Type` dropdown in `DAGWorkspace` to `db_source`, `transform`, and `db_sink` (`source`, `filter`, `sink` temporarily disabled in UI).
- Added `hx-get="/pipelines/{id}/stages/config-fields"` on stage type change to dynamically swap `#stage-config-fields`.
- Rendered type-specific inputs:
  - `db_source`: `driver`, `dsn`, `query`, `columns` (comma-separated).
  - `transform`: `rename` (JSON map), `set` (JSON map).
  - `db_sink`: `driver`, `dsn`, `table`, `columns` (pre-filled with transformed columns from upstream `db_source` + `transform` stages).
- Displayed column chips (`.col-chip`) on DAG stage cards for `db_source`, `transform` (`old -> new`, `+key`), and `db_sink`.

### 2. Backend (`internal/stage/db.go`, `internal/pipeline/dag_handler.go`, `internal/server/server.go`)
- Added optional `Columns []string` to `stage.DBSourceConfig`.
- Added `GET /pipelines/{id}/stages/config-fields` (`GetStageConfigFields`) to render `StageConfigFields` with computed transformed columns (`collectTransformedColumns`).
- Updated `AddNode` to assemble JSON config from structured form inputs and validate via `stage.NewDefaultRegistry().Create(...)`.

### 3. Unit Tests
- `web/components/stage_config_test.go`: Tests rendering `StageConfigFields` for `db_source`, `transform`, and `db_sink`.
- `internal/pipeline/dag_handler_test.go`: Added `TestHandler_StageConfigFieldsAndTransformedColumns`.
