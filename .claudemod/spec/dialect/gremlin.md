# Specification: Gremlin Dialect

## 1. Goal

Provide Apache TinkerPop-compatible graph database support through Gremlin traversal queries, enabling ent schemas to work with graph databases like JanusGraph and Amazon Neptune.

## 2. User Stories

- **As a developer**, I want ent to generate Gremlin traversals for my graph database.
- **As a developer**, I want to connect via WebSocket or HTTP to a Gremlin server.

## 3. Technical Requirements

- **Driver** (`dialect/gremlin/driver.go`, ~59 lines):
  - Wraps Gremlin `Client`.
  - `Exec()` and `Query()` both send Gremlin traversal queries.
  - Args format: `dsl.Bindings` (map of variable bindings).
  - Results format: `*gremlin.Response`.

- **Client** (`dialect/gremlin/`):
  - `Transport` interface for HTTP and WebSocket.
  - GraphSON encoding/decoding (`encoding/graphson/`).
  - WebSocket connection management (`internal/ws/`).

- **DSL** (`dialect/gremlin/graph/dsl/`):
  - Traversal builder: `g.V()`, `g.E()`.
  - Filter predicates: `.Has()`, `.HasLabel()`, `.HasNot()`.
  - Traversal steps: `.Out()`, `.In()`, `.Both()`, `.Values()`.
  - Compiles to Groovy scripts with variable bindings.

- **SchemaMode**: `Unique` only (no Indexes, Cascade, or Migrate support).

- **Templates** (`entc/gen/template/dialect/gremlin/`, 13 templates):
  - Generate Gremlin traversal code instead of SQL.
  - Result parsing from GraphSON format.

## 4. Acceptance Criteria

- **Scenario**: Query users with Gremlin
  - **Given** a User entity with Gremlin storage
  - **When** `client.User.Query().Where(user.NameEQ("alice")).All(ctx)` is called
  - **Then** a Gremlin traversal is executed against the graph database.

## 5. Edge Cases

- GraphSON v1/v2/v3 compatibility.
- WebSocket reconnection on connection loss.
- Gremlin servers with different property graph models.
- No migration support - schema must be pre-configured.
