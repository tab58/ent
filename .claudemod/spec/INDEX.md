# Project: ent (Entity Framework for Go)

## 1. Overview

ent is a Go entity framework that enables developers to define database schemas as Go code and automatically generates type-safe database clients via code generation. Originally developed by Meta (Facebook), now maintained by the Atlas team. This fork is maintained for integration into a larger application with the goal of adding a Neo4j (Cypher) dialect.

**Module:** `entgo.io/ent`
**Go Version:** 1.24+
**License:** Apache 2.0

## 2. System Components

- **Schema Definition API** (`schema/`): Builder-pattern API for defining entities, fields, edges, indexes, and mixins.
- **Code Generation Engine** (`entc/`): Loads schemas, builds an entity graph, and executes Go templates to generate type-safe query builders and CRUD operations.
- **Database Dialects** (`dialect/`): Pluggable backend support via the `dialect.Driver` interface. Currently supports SQL (MySQL, PostgreSQL, SQLite), Gremlin (Apache TinkerPop), and Neo4j (Cypher).
- **Query Language** (`entql/`): Dynamic query expression system for runtime predicate evaluation.
- **Privacy Framework** (`privacy/`): Fine-grained access control with Allow/Deny/Skip decision model.
- **CLI Tools** (`cmd/`): Command-line tools for schema management, code generation, and schema fixing.

## 3. Data Flow

1. Developer defines schemas as Go structs implementing `ent.Schema`.
2. `entc/load` parses Go AST to extract schema definitions (fields, edges, indexes).
3. `entc/gen` builds an in-memory entity graph (`*gen.Graph`).
4. Code generation selects dialect-specific templates based on configured `Storage`.
5. Templates generate type-safe Go packages: entity types, query builders, mutations, client.
6. At runtime, generated code uses `dialect.Driver` to execute queries against the database.

## 4. Technology Stack

- **Language:** Go 1.24+
- **Schema Migration:** ariga.io/atlas
- **CLI Framework:** github.com/spf13/cobra
- **AST Parsing:** golang.org/x/tools
- **Observability:** go.opencensus.io
- **JSON:** github.com/json-iterator/go

## 5. Architecture Decisions

- **Schema-as-Code:** Schemas are Go structs, not YAML/JSON/SQL. Enables IDE support, type safety, and composability via mixins.
- **Code Generation over Reflection:** Type-safe generated code instead of runtime reflection. Catches errors at compile time.
- **Pluggable Dialects:** `dialect.Driver` interface allows multiple backend databases. Storage drivers registered in `entc/gen/storage.go`.
- **Template-based Generation:** Go text templates allow customization and extension by users.
- **Plugin Architecture:** `entc.Extension` interface for third-party integrations with hooks for pre/post-generation.

## 6. Feature Specifications

- [./core/schema.md](Core Schema Definition)
- [./codegen/engine.md](Code Generation Engine)
- [./codegen/templates.md](Template System)
- [./dialect/driver.md](Dialect Driver Interface)
- [./dialect/sql.md](SQL Dialect)
- [./dialect/gremlin.md](Gremlin Dialect)
- [./dialect/neo4j.md](Neo4j Cypher Dialect)
- [./query/entql.md](Query Language)
- [./privacy/access-control.md](Privacy & Access Control)
- [./cli/tools.md](CLI Tools)
