# Pipeline Repository Implementation

## Overview
Implemented the repository layer for the `Pipeline` domain model in `internal/pipeline` using SQLite via `github.com/mattn/go-sqlite3`.

## Changes

### 1. Repository Interface & Domain Errors (`internal/pipeline/repository.go`)
- Defined domain sentinel errors:
  - `ErrNotFound`: returned when a queried, updated, or deleted pipeline ID does not exist.
  - `ErrAlreadyExists`: returned when attempting to insert a pipeline with a duplicate ID.
- Defined the `Repository` interface:
  - `Create(ctx context.Context, p *Pipeline) error`
  - `GetByID(ctx context.Context, id string) (*Pipeline, error)`
  - `List(ctx context.Context) ([]*Pipeline, error)`
  - `Update(ctx context.Context, p *Pipeline) error`
  - `Delete(ctx context.Context, id string) error`

### 2. SQLite Implementation (`SQLiteRepository`)
- Embedded `*sql.DB` connection with schema initialization on startup.
- Table schema:
  ```sql
  CREATE TABLE IF NOT EXISTS pipelines (
      id TEXT PRIMARY KEY,
      name TEXT NOT NULL,
      nodes TEXT NOT NULL,
      edges TEXT NOT NULL
  );
  ```
- Nodes and edges are stored as JSON strings to maintain fidelity with `json.RawMessage` node configurations.
- Default empty slices `[]Node{}` and `[]Edge{}` are preserved on serialization and deserialization.

### 3. Unit Tests (`internal/pipeline/repository_test.go`)
- Added comprehensive unit tests against an in-memory SQLite database (`:memory:`):
  - `TestSQLiteRepository_CreateAndGetByID`: tests creating pipeline, unique constraint (`ErrAlreadyExists`), retrieving by ID, and `ErrNotFound` on non-existent records.
  - `TestSQLiteRepository_Create_NilPipeline`: guards against nil inputs.
  - `TestSQLiteRepository_List`: verifies empty lists and multiple pipeline ordering.
  - `TestSQLiteRepository_Update`: tests mutating attributes and `ErrNotFound` on non-existent records.
  - `TestSQLiteRepository_Delete`: tests deletion and `ErrNotFound` on non-existent records.
