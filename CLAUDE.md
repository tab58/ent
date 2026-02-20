# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Ent is an entity framework (ORM + code generator) for Go. Users define schemas in Go, and `entc` generates a fully typed client with CRUD operations, graph traversals, and predicates for the chosen storage backend.

Module: `entgo.io/ent` | Go 1.24+ | License: Apache 2.0

## Common Commands

```bash
# Build
go build ./...

# Run all unit tests (what CI runs per-package)
go test -race ./...

# Run tests for a specific package
go test -race ./dialect/neo4j/...
go test -race ./entc/gen/...

# Run a single test
go test -race -run TestDriverExec ./dialect/neo4j/

# Lint (matches CI)
golangci-lint run --verbose

# Code generation (run after changing templates or schema)
go generate ./...

# Verify generated files are up to date (what CI checks)
go generate ./... && git status --porcelain

# Integration tests (requires Docker services — see .github/workflows/ci.yml)
cd entc/integration && go test -race -count=2 ./...
```

CI runs unit tests per-directory: `cmd/`, `dialect/`, `schema/`, `entc/load`, `entc/gen`, `examples/`.

## Architecture

### Code Generation Pipeline

```
User schema (Go structs) → entc/load (AST parse) → entc/gen (Graph build) → text/template → generated .go files
```

1. **`entc/load`** — Parses the user's schema package via `golang.org/x/tools/go/packages`. Produces `load.Schema` structs.
2. **`entc/gen`** — `NewGraph(cfg, schemas...)` builds an in-memory `*gen.Graph` with `Type` nodes and `Edge` connections. `Graph.Gen()` executes templates and writes output files through `goimports`.
3. **`entc/entc.go`** — Public entry point: `Generate()`, `LoadGraph()`. The `Extension` interface lets third-party packages inject hooks, templates, and annotations.

### Dialect System

Each storage backend (SQL, Gremlin, Neo4j) is implemented as:
- **Runtime driver** in `dialect/<name>/` — implements `dialect.Driver` (Exec, Query, Tx, Close, Dialect)
- **Query builder** — `dialect/sql.Selector`, `dialect/gremlin/graph/dsl.Traversal`, `dialect/neo4j/cypher.Builder`
- **Templates** in `entc/gen/template/dialect/<name>/` — generate dialect-specific methods on the client types
- **Storage registration** in `entc/gen/storage.go` — maps dialect name to builder type, imports, schema capabilities, and op codes

Supported dialects: `mysql`, `sqlite3`, `postgres`, `gremlin`, `neo4j`

### Template Structure

Templates live in `entc/gen/template/`. There are two layers:
- **Shared templates** (`base.tmpl`, `client.tmpl`, `where.tmpl`, etc.) — dialect-agnostic client scaffolding
- **Dialect templates** (`template/dialect/<name>/`) — each dialect provides: `create`, `query`, `update`, `delete`, `decode`, `predicate`, `errors`, `meta`, `globals`, `by`, `group`, `select`, `open`

Templates use Go `text/template` with a rich `funcmap` from `entc/gen/func.go`. Each template defines named sub-templates (e.g., `dialect/neo4j/create`) that the shared templates call via `{{- with extend $ "..." }}{{ template "dialect/neo4j/create" . }}{{ end }}`.

### Key Type Hierarchy (entc/gen)

- `Graph` — the top-level container; holds all `Type` nodes and the generation config
- `Type` — represents one entity (e.g., User); contains `Field`s, `Edge`s, `Index`es
- `Field` — a schema field with name, Go type, storage type, validators, defaults
- `Edge` — a relationship between types (O2O, O2M, M2O, M2M); has Rel type, inverse info
- `Storage` — dialect configuration: builder type, supported schema modes, op-code mapping

### Adding a New Dialect

Follow the pattern established by Gremlin and Neo4j:
1. Create `dialect/<name>/` with driver, query builder, config, and response types
2. Add templates in `entc/gen/template/dialect/<name>/` matching the required set (create, query, update, delete, decode, predicate, errors, meta, globals, by, group, select, open)
3. Register a `*Storage` entry in `entc/gen/storage.go` with the builder type, dialect constants, op-code mappings, and schema mode flags
4. Add the dialect constant in `dialect/dialect.go`

### Neo4j Dialect (claudemod-neo4j branch)

The Neo4j dialect maps ent concepts to Cypher:
- Entities → node labels (`:User`)
- Fields → node properties
- Edges → relationships with `SCREAMING_SNAKE_CASE` names (e.g., `USER_HAS_PET`)
- IDs → KSUID strings on the `id` property
- Deletes → always `DETACH DELETE`
- Uniqueness → application-level OPTIONAL MATCH guards (no database constraints yet)
- Transactions → `NopTx` wrapper (real ACID transactions deferred)

Key files: `dialect/neo4j/driver.go` (driver with `queryRunner` interface for mock injection), `dialect/neo4j/cypher/builder.go` (Cypher DSL), `dialect/neo4j/response.go` (result scanning).

## Testing Patterns

- **Table-driven tests** are the standard pattern throughout the codebase
- **Race detection** is always on: `go test -race`
- **Mock injection** via interfaces (e.g., Neo4j's `queryRunner` interface allows pure-Go testing without a live database)
- **Template contract tests** (`entc/gen/neo4j_tmpl_test.go`) verify all required template definitions exist before code generation
- Integration tests in `entc/integration/` run against real databases via Docker services in CI
