# Transform Column Mapping UI

## Overview
Replaced the raw JSON `rename`/`set` textareas in the `transform` stage form with an interactive row-based **Column Mapping UI** for better UX.

## Changes

### 1. Column Mapping UI (`web/components/stage_config.templ`, `web/static/js/dag.js`, `web/static/css/styles.css`)
- Replaced `rename` and `set` JSON textareas with:
  - **Column Mapping (Source → Target)**: Rows of `rename_from[]` → `rename_to[]`, pre-populated from upstream `db_source` columns when adding a new `transform` stage, or from existing mappings when editing.
  - **Static Fields (Field = Value)**: Rows of `set_key[]` = `set_val[]`.
  - **+ Add Mapping** / **+ Add Static Field** buttons and per-row remove buttons.
- Added `MappingPair` struct and `RenamePairs`/`SetPairs` fields on `StageConfigValues`.

### 2. Backend Parsing (`internal/pipeline/dag_handler.go`)
- Updated `buildStageConfigFromForm` to parse `rename_from[]`/`rename_to[]` and `set_key[]`/`set_val[]` (ignoring blank or unchanged target renames, while preserving JSON fallback).
- Updated `extractStageConfigValues` to populate `RenamePairs` and `SetPairs` when editing an existing `transform` node.

### 3. Unit Tests
- `web/components/stage_config_test.go`: Updated `transform` test to verify mapping rows and pre-populated source columns.
- `internal/pipeline/dag_handler_test.go`: Added assertions for `rename_from[]`, `rename_to[]`, `set_key[]`, and `set_val[]` submission.
