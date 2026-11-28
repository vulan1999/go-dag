# Stage Implementation

## Overview
Implemented the stage abstraction, registry, built-in ETL processing stages, and SQLite/SQL database connectors in `internal/stage`.

## Changes

### 1. Core Stage & Registry (`internal/stage/stage.go`)
- Defined `Record` (`map[string]any`) and the `Stage` interface (`Type()`, `Execute(ctx, inputs)`).
- Implemented `Registry` with `NewRegistry()`, `NewDefaultRegistry()`, `Register()`, `Create()`, and `Types()`.
- Defined sentinel errors: `ErrUnknownStageType`, `ErrInvalidConfig`.

### 2. Built-in ETL Stages (`internal/stage/builtins.go`)
- `source`: Emits configured static `records`.
- `filter`: Filters records using `field`, `op` (`eq`, `neq`, `exists`), and `value`.
- `transform`: Renames keys (`rename`) and assigns static fields (`set`).
- `sink`: Writes records as JSON lines and passes them through.

### 3. Database Connector Stages (`internal/stage/db.go`)
- `db_source`: Connects via `database/sql` (defaulting `driver` to `sqlite3`), runs `query`, and outputs `[]Record`.
- `db_sink`: Validates table/column identifiers and inserts incoming `[]Record` into the target table inside a transaction.

### 4. Frontend Datalist (`web/components/dag_canvas.templ`)
- Added `db_source` and `db_sink` suggestions to the stage type input datalist.

### 5. Unit Tests
- `internal/stage/stage_test.go`: Registry defaults, custom registration, and error handling.
- `internal/stage/builtins_test.go`: `source`, `filter`, `transform`, and `sink` execution and cancellation.
- `internal/stage/db_test.go`: End-to-end DB A → filter → transform → DB B ETL test using SQLite files, plus config validation.
