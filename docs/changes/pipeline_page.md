# Pipeline Management Page & Repository-Backed API

## Overview
Implemented a Templ + HTMX web page at `/pipelines` and wired all pipeline CRUD operations (`Create`, `GetByID`, `List`, `Update`, `Delete`) to `SQLiteRepository` in `internal/pipeline/repository.go`.

## Changes

### 1. Frontend Layout, Static CSS, Components & Page (`web/`)
- `web/static/styles.css`: Added stylesheet for layout, cards, forms, tables, buttons, and error messages, served under `/static/`.
- `web/layouts/base.templ`: Included `/static/styles.css`, HTMX (`2.0.4`), and a `Pipelines` navigation link (`/pipelines`).
- `web/components/pipeline.templ`: Added `PipelineView`, `PipelineRow` (display row with Edit/Delete HTMX actions), and `PipelineEditRow` (inline HTMX edit form with Save/Cancel).
- `web/pages/pipelines.templ`: Added `Pipelines` page with a Create Pipeline form (`hx-post="/pipelines"`) and a table of all pipelines.
- Regenerated `*_templ.go` files using `templ generate`.

### 2. Repository-Backed HTTP Handlers (`internal/pipeline/handler.go`)
- Added `Handler` struct and `NewHandler(repo Repository) *Handler`.
- Implemented `List`, `Create`, `GetByID`, `Update`, and `Delete` supporting both JSON API clients and HTMX (`HX-Request: true` / form-encoded payloads).

### 3. Server & Database Wiring (`cmd/server/main.go`, `internal/server/server.go`)
- `cmd/server/main.go`: Opens SQLite via `DB_PATH` (default `flow.db`), initializes `pipeline.NewSQLiteRepository(db)`, and passes it to `server.New`.
- `internal/server/server.go`: Serves `/static/` and registers `GET /pipelines`, `POST /pipelines`, `GET /pipelines/{id}`, `PUT /pipelines/{id}`, and `DELETE /pipelines/{id}`.

### 4. Unit Tests
- `internal/pipeline/handler_test.go`: Added tests for `List`, `Create`, `GetByID`, `Update`, and `Delete` (both JSON and HTMX) against an in-memory SQLite repository.
- `web/components/pipeline_test.go` and `web/pages/pipelines_test.go`: Added rendering tests for the new Templ components and page.
