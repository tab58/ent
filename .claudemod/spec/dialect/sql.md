# Specification: SQL Dialect

## 1. Goal

Provide comprehensive SQL database support for MySQL, PostgreSQL, and SQLite through a shared query builder, schema migration engine, and graph traversal operations.

## 2. User Stories

- **As a developer**, I want ent to generate SQL queries for my PostgreSQL/MySQL/SQLite database.
- **As a developer**, I want automatic schema migrations when my entity schemas change.
- **As a developer**, I want type-safe query builders that prevent SQL injection.

## 3. Technical Requirements

- **Driver** (`dialect/sql/driver.go`, ~246 lines):
  - Wraps `database/sql.DB` via `Conn` struct.
  - Implements `dialect.Driver` with `ExecContext`/`QueryContext` delegation.
  - Transaction support via `sql.Tx` wrapper.

- **Query Builder** (`dialect/sql/builder.go`, ~3200+ lines):
  - `Selector` for SELECT with JOINs, WHERE, GROUP BY, ORDER BY, LIMIT/OFFSET.
  - `InsertBuilder`, `UpdateBuilder`, `DeleteBuilder` for DML.
  - `TableBuilder`, `ColumnBuilder` for DDL.
  - Subquery and CTE support.
  - Predicate builder with all comparison operators.

- **Graph Operations** (`dialect/sql/sqlgraph/`):
  - `QueryNodes`, `QueryEdges` for traversing relationships in SQL.
  - Edge handling: O2O, O2M, M2M with join tables.
  - Cascade delete support.
  - Constraint error detection.

- **Schema Migration** (`dialect/sql/schema/`):
  - Atlas-based migration engine.
  - Dialect-specific DDL: `mysql.go`, `postgres.go`, `sqlite.go`.
  - Diff-based migration (compare desired vs actual schema).
  - Concurrent migration support (without table copying).

- **JSON Support** (`dialect/sql/sqljson/`):
  - JSON field querying for MySQL, PostgreSQL, SQLite.
  - Path-based JSON access.

## 4. Acceptance Criteria

- **Scenario**: Query users with SQL
  - **Given** a User entity with SQL storage
  - **When** `client.User.Query().Where(user.NameEQ("alice")).All(ctx)` is called
  - **Then** a parameterized SQL SELECT is executed against the database.

## 5. Edge Cases

- Dialect-specific SQL syntax differences (e.g., LIMIT vs TOP).
- JSON field support varies by database engine.
- Schema migration with existing data must preserve records.
