# Specification: Neo4j Driver

## 1. Purpose

Implements the `dialect.Driver` interface for Neo4j graph database. Provides connection management, query execution (read/write routing), configuration parsing, and result scanning. This is the runtime layer that generated Neo4j client code calls to interact with the database.

## 2. Key Components

- `dialect/neo4j/driver.go` — `Driver` struct implementing `dialect.Driver`; `queryRunner` interface for DI/testing; `neo4jRunner` production implementation wrapping `neo4j-go-driver/v6`
- `dialect/neo4j/config.go` — `Config` struct (URI, Username, Password, Database); `ParseURI()` for connection string parsing; `Build()` factory method
- `dialect/neo4j/response.go` — `Response` wrapper over `[]*neo4j.Record`; typed readers (`ReadInt`, `ReadBool`, `ReadNodeMaps`, `ReadSingle`); `Scan()` via JSON round-trip

## 3. Data Models

- **Config** — `{URI string, Username string, Password string, Database string}` — Connection parameters. Database defaults to `"neo4j"`.
- **Driver** — `{runner queryRunner, database string}` — Stateful driver wrapping a connection.
- **Response** — `{records []*ndriver.Record, columns []string}` — Collected query results with typed extraction methods.
- **queryRunner** (interface) — `{executeRead(ctx, db, query, params), executeWrite(ctx, db, query, params), close(ctx)}` — Abstraction over Neo4j session management.

## 4. Interfaces

- **dialect.Driver** (implemented by `Driver`):
  - `Exec(ctx, query, args, v)` — Write operations; args must be `map[string]any`, v must be `*Response`
  - `Query(ctx, query, args, v)` — Read operations; same arg/result constraints
  - `Tx(ctx) (dialect.Tx, error)` — Returns `NopTx` (real transactions deferred)
  - `Close() error` — Closes the underlying Neo4j driver
  - `Dialect() string` — Returns `"neo4j"`
- **Config.Build()** — Factory: creates `neo4j.Driver` via `ndriver.NewDriver()`, wraps in `Driver`
- **ParseURI(uri)** — Parses `bolt://user:pass@host:7687/dbname` into `Config`

## 5. Dependencies

- **Depends on:** `github.com/neo4j/neo4j-go-driver/v6` (neo4j.Driver, Session, ManagedTransaction, Record), `entgo.io/ent/dialect` (Driver interface, NopTx)
- **Depended on by:** Generated client code (via `Exec`/`Query`), `Config.Build()` from generated `Open()` method

## 6. Acceptance Criteria

- `Driver` satisfies `dialect.Driver` at compile time (`var _ dialect.Driver = (*Driver)(nil)`)
- `Exec` routes through `executeWrite`; `Query` routes through `executeRead`
- Args validated as `map[string]any`; result validated as `*Response`
- `ParseURI` handles bolt://, neo4j:// schemes with optional user/pass/dbname
- `Response.Scan` decodes arbitrary record shapes via JSON round-trip
- `Response.ReadInt` handles Neo4j `int64` -> Go `int` conversion
- All methods return wrapped errors with `"neo4j: "` prefix

## 7. Edge Cases

- `Exec`/`Query` with wrong arg types -> descriptive error with actual type name
- `ParseURI("")` -> `"neo4j: empty URI"` error
- `ParseURI` with missing host -> `"neo4j: invalid URI"` error
- `ParseURI` with no database path -> defaults to `"neo4j"`
- `Response.ReadInt` on empty records -> `"neo4j: no records in response"`
- `Response.ReadNodeMaps` on nil records -> `"neo4j: nil records in response"`
- `Close()` on nil runner -> `"neo4j: driver connection is nil"`

## 8. Production Gaps

- **Transactions:** `Tx()` returns `NopTx` — no real ACID transaction support yet. Generated code can call `Tx()` but operations are auto-committed per-statement.
- **Connection pooling:** Delegated entirely to `neo4j-go-driver`; no ent-level pool management.
- **Context cancellation:** Sessions created per-call with `defer Close`; no long-lived session reuse.
- **Retry logic:** No built-in retry for transient failures (e.g., leader election during cluster failover).
