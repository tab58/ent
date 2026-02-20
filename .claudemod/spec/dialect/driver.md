# Specification: Dialect Driver Interface

## 1. Goal

Define a minimal, pluggable interface for database backend drivers, enabling ent to support multiple database technologies through a common abstraction.

## 2. User Stories

- **As a framework maintainer**, I want a minimal driver interface so new databases can be added with low coupling.
- **As a dialect author**, I want clear contracts for implementing Exec, Query, and transaction support.

## 3. Technical Requirements

- **Core Interfaces** (`dialect/dialect.go`):

```go
type ExecQuerier interface {
    Exec(ctx context.Context, query string, args, v any) error
    Query(ctx context.Context, query string, args, v any) error
}

type Driver interface {
    ExecQuerier
    Tx(context.Context) (Tx, error)
    Close() error
    Dialect() string
}

type Tx interface {
    ExecQuerier
    Commit() error
    Rollback() error
}
```

- **Polymorphic Arguments/Results**:
  - `args` and `v` are `any` type, allowing each dialect to define conventions:
    - SQL: `args = []any`, `v = *sql.Rows | *sql.Result`
    - Gremlin: `args = dsl.Bindings (map)`, `v = *gremlin.Response`
  - Generated code type-asserts based on selected dialect.

- **Dialect Constants**:
  - `dialect.MySQL`, `dialect.SQLite`, `dialect.Postgres`, `dialect.Gremlin`
  - New dialects add constants here.

- **Storage Registration** (`entc/gen/storage.go`):
  - `*Storage` struct defines: Name, Builder type, Dialects, SchemaMode, Ops, OpCode, Init.
  - SchemaMode bitmask: `Unique | Indexes | Cascade | Migrate`.
  - Predicate operations mapped to dialect-specific names via `OpCode`.

## 4. Acceptance Criteria

- **Scenario**: Implement a new dialect driver
  - **Given** a type implementing `dialect.Driver`
  - **When** registered in `storage.go` with templates
  - **Then** `entc generate --storage <name>` produces working generated code.

## 5. Edge Cases

- Driver must handle context cancellation gracefully.
- Transaction rollback on panic must be supported.
- `NopTx()` available for dialects without transaction support.
