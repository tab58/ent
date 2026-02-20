# Changelog

## 2026-02-19 — Neo4j (Cypher) Dialect Implementation

### Summary

Added complete Neo4j graph database support to the ent entity framework. The implementation includes a dialect driver, Cypher query builder, storage registration, and 13 code generation templates mirroring the Gremlin template set.

### Implementation

**Dialect constant** — Added `dialect.Neo4j = "neo4j"` to `dialect/dialect.go`.

**Cypher query builder** (`dialect/neo4j/cypher/`) — `Builder` struct with clause methods (Match, Where, Create, Merge, Set, Remove, Delete, DetachDelete, Return, OrderBy, Skip, Limit), parameter management (AddParam, SetParam), predicate composition support (WhereClauses, Params, CollectWhere), Query() output, and Clone() for deep copy.

**Neo4j driver** (`dialect/neo4j/`) — `Driver` implementing `dialect.Driver` with `queryRunner` interface for testability. `Config` struct with `Build()` factory and `ParseURI()` helper. `Response` struct with typed read methods (ReadInt, ReadBool, ReadNodeMaps, ReadSingle, Scan).

**Storage registration** (`entc/gen/storage.go`) — Neo4j storage entry with Builder type, imports, SchemaMode (Unique | Indexes | Cascade), and OpCode mapping including EqualFold/ContainsFold.

**Templates** (`entc/gen/template/dialect/neo4j/`) — 13 templates: open, create, update, delete, query, decode, predicate, errors, meta, globals, group, select, by.

### Code Review Fixes

- Fixed group/select return type mismatch by adding `Response.Scan()` method using JSON round-trip decoding.
- Fixed `query.tmpl` dead initializer in `neo4jQuery` with if/else restructuring.
- Fixed predicate AND/OR/NOT double-WHERE and param collision by adding `Builder.CollectWhere()` method.
- Removed unused `Option` type and `options` struct from `config.go`, simplified `Build()` signature.
- Added bounds check on empty `Values` slice in `ReadInt` and `ReadBool`.
- Confirmed `DetachDelete` is always correct since Neo4j storage permanently includes Cascade.

### Spec Updates

- Updated neo4j.md to reflect: `queryRunner` interface, `NewDriver` constructor, simplified `Build()` signature, `CollectWhere`/`WhereClauses`/`Params` methods, `Response.Scan`, `EXISTS { ... }` predicate pattern, `EqualFold`/`ContainsFold` ops, bidi edge support, always-DetachDelete behavior.
- Removed phantom specs: `Option` type, conditional DELETE path.
- Added new acceptance criterion for composed predicates (AND/OR/NOT).
- Added edge cases for `ReadInt`/`ReadBool` bounds checks and `Scan` JSON round-trip precision.

### Completed Tasks

- [x] Add `Neo4j` dialect constant
- [x] Create Cypher query builder (`dialect/neo4j/cypher/`)
- [x] Create Neo4j driver package (`dialect/neo4j/`)
- [x] Register Neo4j storage in `entc/gen/storage.go`
- [x] `create.tmpl`
- [x] `query.tmpl`
- [x] `update.tmpl`
- [x] `delete.tmpl`
- [x] `predicate.tmpl`
- [x] `decode.tmpl`
- [x] `open.tmpl`
- [x] `errors.tmpl`
- [x] `meta.tmpl`
- [x] `by.tmpl`
- [x] `globals.tmpl`
- [x] `group.tmpl`
- [x] `select.tmpl`
- [x] Fix `group.tmpl` return type mismatch (added `Response.Scan`)
- [x] Fix `select.tmpl` return type mismatch (use `Response.Scan`)
- [x] Fix `query.tmpl` dead initializer in `neo4jQuery`
- [x] Fix `predicate.tmpl` AND/OR/NOT double-WHERE (added `CollectWhere`)
- [x] Remove unused `options` struct in `config.go`
- [x] Add bounds check on `Values` slice in `response.go`
- [x] Conditional `DetachDelete` confirmed correct as-is
