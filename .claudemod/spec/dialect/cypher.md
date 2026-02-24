# Specification: Cypher Query Builder

## 1. Purpose

Provides a declarative, fluent DSL for assembling parameterized Cypher queries. Equivalent to `sql.Selector` for SQL and `dsl.Traversal` for Gremlin. Used by generated Neo4j client code to construct MATCH, CREATE, SET, DELETE, and RETURN statements with safe parameter binding.

## 2. Key Components

- `dialect/neo4j/cypher/builder.go` — `Builder` struct with fluent methods for all Cypher clauses; parameter management (`AddParam`, `SetParam`); `CollectWhere` for predicate combinators; `Clone()` for deep copy; `Query()` to produce final `(string, map[string]any)`

## 3. Data Models

- **Builder** — Core struct holding clause slices and parameter state:
  - `match []string` — MATCH patterns (e.g., `(n:User)`)
  - `where []string` — WHERE conditions joined by AND
  - `create []string` — CREATE patterns
  - `merge []string` — MERGE patterns
  - `set []string` — SET expressions (joined by comma)
  - `remove []string` — REMOVE expressions
  - `del []string` — DELETE/DETACH DELETE expressions
  - `ret []string` — RETURN expressions
  - `orderBy []string` — ORDER BY expressions
  - `skip *int`, `limit *int` — Pagination
  - `params map[string]any` — Parameter map (`$pN` -> value)
  - `paramN int` — Auto-increment counter for anonymous params

## 4. Interfaces

- **Fluent API** (all return `*Builder` for chaining):
  - `Match(pattern)`, `Where(cond)`, `Create(pattern)`, `Merge(pattern)`
  - `Set(expr)`, `Remove(expr)`, `Delete(expr)`, `DetachDelete(expr)`
  - `Return(exprs...)`, `OrderBy(expr)`, `Skip(n)`, `Limit(n)`
- **Parameter Management:**
  - `AddParam(value) string` — Returns `$pN` placeholder, auto-increments counter
  - `SetParam(name, value)` — Sets named parameter
  - `Params() map[string]any` — Returns parameter map (for predicate combinators)
- **Predicate Support:**
  - `WhereClauses() []string` — Raw WHERE conditions (for AND/OR/NOT extraction)
  - `CollectWhere(fn func(*Builder)) []string` — Captures WHERE clauses added by fn, removes them from builder, preserves params
- **Output:**
  - `Query() (string, map[string]any)` — Assembles full Cypher string + params
  - `Clone() *Builder` — Deep copy (all slices, params, pointers)

## 5. Dependencies

- **Depends on:** Standard library only (`fmt`, `maps`, `strings`)
- **Depended on by:** Generated Neo4j templates (create, query, update, delete, predicate, by, group, select), `entc/gen/storage.go` (Builder type registration)

## 6. Acceptance Criteria

- Clause ordering matches Cypher spec: MATCH -> WHERE -> CREATE -> MERGE -> SET -> REMOVE -> DELETE -> RETURN -> ORDER BY -> SKIP -> LIMIT
- Multiple MATCH clauses emit separate `MATCH` keywords (not comma-joined)
- WHERE conditions joined by `AND`
- SET expressions joined by comma
- `DetachDelete` emits `DETACH DELETE` (not just `DELETE`)
- `AddParam` produces sequential `$p0`, `$p1`, `$p2` placeholders
- `CollectWhere` captures and removes WHERE clauses without affecting params
- `Clone` produces independent deep copy (modifying clone doesn't affect original)
- Empty builder produces empty string

## 7. Edge Cases

- `Clone(nil)` returns nil (nil-safe)
- Builder with only RETURN clause (no MATCH) -> valid Cypher for count queries
- `CollectWhere` on builder with existing WHERE clauses -> only captures new ones
- Parameters with same value get distinct names (`$p0`, `$p1`)
- `Skip(0)` emits `SKIP 0` (explicit zero is valid Cypher)

## 8. Production Gaps

- **WITH clause:** Not supported — needed for multi-part queries and subqueries
- **UNWIND clause:** Not supported — needed for batch operations
- **OPTIONAL MATCH:** Must be manually composed as `Match("OPTIONAL ...")` string — no first-class support
- **Raw Cypher injection:** No escaping or validation of clause content — relies on generated templates to produce safe patterns
- **Query validation:** No structural validation before `Query()` — invalid clause combinations silently produce invalid Cypher
