# Project: Ent — Entity Framework for Go

## 1. Overview

Ent is an entity framework (ORM + code generator) for Go. Users define schemas as Go structs, and `entc` generates a fully typed client with CRUD operations, graph traversals, and predicates for the chosen storage backend. This branch (`claudemod-neo4j`) adds a **Neo4j graph database dialect** targeting production readiness.

Module: `entgo.io/ent` | Go 1.24+ | License: Apache 2.0

## 2. Technology Stack

- **Language:** Go 1.24
- **Code Generation:** `text/template` with custom funcmap
- **Schema Parsing:** `golang.org/x/tools/go/packages` (AST-based)
- **SQL Backend:** `database/sql` (MySQL, PostgreSQL, SQLite)
- **Gremlin Backend:** Custom HTTP/WebSocket client
- **Neo4j Backend:** `github.com/neo4j/neo4j-go-driver/v6`
- **CLI:** `github.com/spf13/cobra`
- **ID Generation:** KSUID (Neo4j), auto-increment (SQL)

## 3. Entry Points

- `entc/entc.go` — Public API: `Generate()`, `LoadGraph()`, Extension interface
- `cmd/ent/ent.go` — CLI: `init`, `generate`, `describe` commands
- `ent.go` — Schema interface definitions (`Schema`, `Mixin`, `Policy`)

## 4. Directory Structure

```
entgo.io/ent/
  cmd/ent/           CLI entry point (cobra commands)
  dialect/
    sql/             SQL driver, query builder, sqlgraph traversals
    entsql/          SQL-specific schema annotations
    gremlin/         Gremlin driver, DSL traversal builder
    neo4j/           Neo4j driver, config, response scanning
      cypher/        Cypher query builder DSL
  entc/
    entc.go          Public codegen API (Generate, LoadGraph)
    gen/             Graph builder, template engine, storage drivers
      template/      Shared templates (client, where, builder, etc.)
        dialect/
          sql/       SQL-specific templates
          gremlin/   Gremlin-specific templates
          neo4j/     Neo4j-specific templates (13 files)
    load/            Schema AST parser (go/packages)
    integration/     Integration tests (Docker-backed)
  entql/             Query language abstraction
  schema/            Schema building blocks (field, edge, index, mixin)
  privacy/           Privacy/authorization policy engine
  examples/          Example projects
```

## 5. Data Flow

### Code Generation Pipeline

```
User schema (Go structs)
  -> entc/load (AST parse via go/packages)
  -> entc/gen (Graph build: Types, Edges, Fields)
  -> text/template execution (shared + dialect-specific)
  -> goimports formatting
  -> Generated .go files
```

### Runtime Query Flow (Neo4j)

```
Generated client method (e.g. UserCreate)
  -> cypher.Builder assembles Cypher query
  -> Driver.Exec/Query dispatches to Neo4j session
  -> neo4jRunner.executeWrite/executeRead
  -> Response wraps neo4j.Record results
  -> JSON round-trip decoding into Go structs
```

## 6. Design Patterns

- **Code Generation** — Schema-first: define once, generate typed CRUD + traversals
- **Dialect Abstraction** — `dialect.Driver` interface with pluggable backends (SQL, Gremlin, Neo4j)
- **Template Layering** — Shared templates call dialect-specific sub-templates via `{{ template "dialect/neo4j/create" }}`
- **Graph Model** — In-memory `Graph` of `Type` nodes with `Edge` connections drives all code generation
- **DI via Interfaces** — `queryRunner` for Neo4j testing, `dialect.Driver` for backend swapping

## 7. External Integrations

- **Neo4j Database** — via `neo4j-go-driver/v6` (bolt protocol)
- **SQL Databases** — MySQL, PostgreSQL, SQLite via `database/sql`
- **Atlas** — Schema migration engine (SQL only, `ariga.io/atlas`)
- **Gremlin Server** — Apache TinkerPop graph database

## 8. Build & Run

```bash
go build ./...                     # Build all packages
go test -race ./...                # Run all unit tests
go test -race ./dialect/neo4j/...  # Neo4j dialect tests
go generate ./...                  # Regenerate code from templates
golangci-lint run --verbose        # Lint
```

## 9. Domain Specifications

### Deep Specs (Neo4j + Codegen)

- [Neo4j Driver](./dialect/neo4j.md) — Runtime driver, config, response scanning
- [Cypher Builder](./dialect/cypher.md) — Cypher query DSL
- [Codegen Engine](./codegen/engine.md) — Graph builder, storage registration, template execution
- [Codegen Templates](./codegen/templates.md) — Neo4j template structure and contracts

### Reference Specs (Other Domains)

- [Core Schema](./core/schema.md) — Schema building blocks
- [CLI Tools](./cli/tools.md) — Command-line interface
- [EntQL](./query/entql.md) — Query language abstraction
- [Privacy](./privacy/access-control.md) — Access control policies
- [SQL Dialect](./dialect/sql.md) — SQL backend reference
- [Gremlin Dialect](./dialect/gremlin.md) — Gremlin backend reference

## 10. Domain Relationships

- Codegen Engine -> Core Schema (parses schema definitions into Graph)
- Codegen Templates -> Codegen Engine (templates executed by Graph.Gen())
- Neo4j Driver -> Codegen Templates (generated code calls driver at runtime)
- Cypher Builder -> Neo4j Driver (builder produces queries for driver execution)
- CLI Tools -> Codegen Engine (CLI invokes Generate/LoadGraph)
- EntQL -> Codegen Engine (query predicates map to dialect operations)
- Privacy -> Codegen Engine (privacy policies injected via Extensions)
