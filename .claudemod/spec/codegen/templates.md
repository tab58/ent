# Specification: Neo4j Code Generation Templates

## 1. Purpose

Dialect-specific templates that generate Neo4j CRUD operations, predicates, and utility methods on the ent client types. Called by shared templates via `{{ template "dialect/neo4j/<name>" . }}`. Each template produces Go methods that use the Cypher builder and Neo4j driver at runtime.

## 2. Key Components

All templates in `entc/gen/template/dialect/neo4j/`:

| Template | Purpose | Generated Methods |
|----------|---------|-------------------|
| `create.tmpl` | Node creation with KSUID IDs, uniqueness guards | `neo4jSave(ctx)`, `neo4j()` |
| `query.tmpl` | MATCH queries with WHERE, pagination, edge traversal | `neo4jAll(ctx)`, `neo4jCount(ctx)`, `neo4jQuery(ctx)` |
| `update.tmpl` | SET/REMOVE operations, edge mutations | `neo4jSave(ctx)` on UpdateOne/Update |
| `delete.tmpl` | DELETE/DETACH DELETE | `neo4jExec(ctx)` |
| `predicate.tmpl` | WHERE conditions for id, fields, edges | Per-field predicate functions |
| `decode.tmpl` | Node property extraction, JSON decoding | `FromResponse(res)` |
| `by.tmpl` | ORDER BY clause builders | `OrderFunc`, field ordering |
| `group.tmpl` | GROUP BY with aggregation | `AggregateFunc`, `neo4jScan(ctx)` |
| `select.tmpl` | Field-selective RETURN | `neo4jScan(ctx)` for select |
| `errors.tmpl` | Constraint violation errors | `NewErrUniqueField`, `NewErrUniqueEdge` |
| `meta.tmpl` | Relationship type constants | `SCREAMING_SNAKE_CASE` labels |
| `globals.tmpl` | Type aliases for driver API uniformity | `queryHook` type alias |
| `open.tmpl` | Client.Open() Neo4j case | `ParseURI` + `cfg.Build()` |

## 3. Template Contract

All 13 templates must define specific named sub-templates. The contract is verified by `TestNeo4jTemplateDefinitions` in `entc/gen/neo4j_tmpl_test.go`.

Required template definitions (58 total across all files):
- `create.tmpl`: `dialect/neo4j/create`, `dialect/neo4j/create/fields`
- `query.tmpl`: `dialect/neo4j/query`, `dialect/neo4j/query/all`, `dialect/neo4j/query/count`
- `update.tmpl`: `dialect/neo4j/update`, `dialect/neo4j/update/fields`, `dialect/neo4j/update/edges`
- `delete.tmpl`: `dialect/neo4j/delete`
- `predicate.tmpl`: `dialect/neo4j/predicate/id`, `dialect/neo4j/predicate/field`, `dialect/neo4j/predicate/edge`
- `decode.tmpl`: `dialect/neo4j/decode`, `dialect/neo4j/decode/one`
- And similar for by, group, select, errors, meta, globals, open

## 4. Key Patterns

### Entity-to-Cypher Mapping
- **Entities** -> Node labels: `:User`, `:Pet`
- **Fields** -> Node properties: `n.name`, `n.age`
- **Edges** -> Relationships: `-[:USER_HAS_PET]->`, `-[:USER_HAS_FRIEND]->`
- **IDs** -> KSUID strings on `id` property

### Create Pattern
```cypher
OPTIONAL MATCH (n:User) WHERE n.name = $p0
WITH count(n) AS existing
WHERE existing = 0
CREATE (n:User {id: $p1, name: $p0, age: $p2})
RETURN n {.*}
```
- KSUID generated for `id`
- OPTIONAL MATCH + WHERE existing = 0 for uniqueness guard
- Zero-row result -> `ErrUniqueField`

### Query Pattern
```cypher
MATCH (n:User)
WHERE n.age > $p0
RETURN n {.*}
ORDER BY n.name ASC
SKIP 10
LIMIT 5
```

### Update Pattern
```cypher
MATCH (n:User)
WHERE n.id = $p0
SET n.name = $p1, n.age = $p2
REMOVE n.optional_field
RETURN n {.*}
```

### Delete Pattern
```cypher
MATCH (n:User)
WHERE n.id = $p0
DETACH DELETE n
```

## 5. Dependencies

- **Depends on:** `entc/gen/template.go` (template loading), `entc/gen/func.go` (funcmap), `entc/gen/type.go` (Type/Field/Edge context)
- **Depended on by:** `entc/gen/graph.go` (Graph.Gen() executes templates)

## 6. Acceptance Criteria

- All 13 template files parse without errors
- All 58 required template definitions present (verified by contract test)
- Template parity with Gremlin dialect (same set of definitions)
- Generated code compiles with `go build`
- Generated CRUD methods produce correct Cypher via `cypher.Builder`
- Predicate functions cover all field types and operations
- Edge traversal generates correct relationship patterns

## 7. Edge Cases

- Entity with no unique fields -> no OPTIONAL MATCH guard in create
- Entity with only edges (no fields) -> CREATE with just id property
- M2M edges -> intermediate relationship nodes (no join tables in Neo4j)
- Optional/Nillable fields -> REMOVE instead of SET to null
- Enum fields -> stored as strings, validated in Go code

## 8. Test Coverage

- `entc/gen/neo4j_tmpl_test.go` — Template contract tests:
  - `TestNeo4jTemplateDefinitions` — All 58 definitions exist
  - `TestNeo4jTemplateParity` — Matches Gremlin template set
  - `TestNeo4jCreateTemplate` — KSUID, uniqueness, edge creation
  - `TestNeo4jQueryTemplate` — MATCH, WHERE, pagination, traversal
  - `TestNeo4jPredicateTemplates` — WHERE conditions, OpCode ops
  - `TestNeo4jCodeGen_WithNeo4jStorage` — Full schema -> codegen integration
