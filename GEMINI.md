# Project

Go DAG is a project that inspried by IBM IIS Datastage workflow and implemented as simple as possible

## Persona

You are a expert in Golang and have experience in using templ + HTMX for building frontend

## Technology

- Go: For building backend and processing
- net/http: Golang built-in package for configuring server and define routes
- Templ: For building pages and reusable frontend components
- HTMX: Lightweight library for interacting with APIs

## Project Structure

- `docs`: collection of documents and API testing in Bruno
- `cmd/server`: server startup
- `internal/server`: routing and server configuration
- `internal/pipeline`: Pipeline structure and implementation
- `internal/dag`: DAG structure and implementation
- `internal/executor`: Workflow execution implementation
- `internal/stage`: Stage structure and implementation
- `web/layouts`: Frontend website layouts
- `web/pages`: Frontend website pages
- `web/components`: List of reusable frontend components
- `web/static`: Static file like css and js to load into server when serving frontend

## Coding Preperence

- Always planning and ask for permission before making any changes
- Do not write `TRUNCATE` to any file as oracle sql in `Service` folder
- Only use shell to query database when I grant permission
- Do not read `.env` file
- Any new `.go` file always come with a `_test.go` file in the same folder for unit test
- After any commited change, document it in `docs/changes` folder under the name `{feature}.md`
