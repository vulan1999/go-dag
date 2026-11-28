# SQL Query Auto-Fill for Output Columns

## Overview
Implemented automatic extraction of output columns from `db_source` SQL queries, supporting simple queries, CTEs (`WITH ...`), scalar subqueries, nested `FROM (...)` subqueries, and `LIMIT 0` SQLite introspection for `SELECT *`.

## Changes

### 1. SQL Projection Parser & Introspection (`internal/stage/sql_parse.go`)
- `ParseSelectColumns(query string) []string`: Tracks parenthesis depth and quotes to locate the outermost `SELECT ... FROM` projection, handle aliases (`AS` and implicit), strip table prefixes, and unwrap `SELECT * FROM (SELECT ...)`.
- `IntrospectQueryColumns(ctx, driver, dsn, query)`: Runs `SELECT * FROM (<query>) LIMIT 0` via `database/sql` when `ParseSelectColumns` returns empty (e.g., `SELECT *` on a table) and the SQLite DB file exists.

### 2. HTMX Auto-Fill UI & Handler (`web/components/stage_config.templ`, `internal/pipeline/dag_handler.go`, `internal/server/server.go`)
- Added `OutputColumnsInput` partial and wired `#cfg-query` to `GET /stages/sql-columns` on input/change.
- Added `GetSQLColumns` handler and automatic column inference in `buildStageConfigFromForm` when `columns` is omitted on `db_source` save.

### 3. Unit Tests
- `internal/stage/sql_parse_test.go`: Tests simple selects, `DISTINCT`, table prefixes, function calls with commas, scalar subqueries, nested `FROM` subqueries, `SELECT * FROM (SELECT ...)`, CTEs, and `IntrospectQueryColumns`.
- `internal/pipeline/dag_handler_test.go`: Added `TestHandler_GetSQLColumns_And_AutoFillOnAdd`.
