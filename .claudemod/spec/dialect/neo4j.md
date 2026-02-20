# Specification: Neo4j (Cypher) Dialect

## 1. Goal

Provide Neo4j graph database support through Cypher queries, enabling ent schemas to map entities to Neo4j nodes (single label) and edges to Neo4j relationships, using `neo4j-go-driver/v6` over the Bolt protocol.

## 2. User Stories

- **As a developer**, I want ent to generate Cypher queries for my Neo4j database so I get type-safe CRUD operations.
- **As a developer**, I want to connect to Neo4j via Bolt URI and authenticate with username/password.
- **As a developer**, I want ent edges to map to Neo4j relationships with directional semantics (owner → inverse).
- **As a developer**, I want KSUID node IDs stored on the `id` property of every node.

## 3. Technical Requirements

### 3.1 Dialect Constant (`dialect/dialect.go`)

```go
const Neo4j = "neo4j"
```

Added alongside existing MySQL, SQLite, Postgres, Gremlin constants.

### 3.2 Driver (`dialect/neo4j/`)

#### 3.2.1 Driver (`driver.go`)

Implements `dialect.Driver`. Wraps `neo4j.Driver` from `github.com/neo4j/neo4j-go-driver/v6/neo4j` through an internal `queryRunner` interface for testability.

**Internal interfaces:**

```go
// queryRunner abstracts Neo4j session management for testability.
type queryRunner interface {
    executeRead(ctx context.Context, database, query string, params map[string]any) ([]*neo4j.Record, error)
    executeWrite(ctx context.Context, database, query string, params map[string]any) ([]*neo4j.Record, error)
    close(ctx context.Context) error
}

// neo4jRunner is the production queryRunner backed by a real neo4j.Driver.
type neo4jRunner struct {
    db neo4j.Driver
}
```

The `queryRunner` interface enables unit testing without a live Neo4j instance by allowing test doubles to be injected in place of the real driver.

**Driver struct:**

```go
type Driver struct {
    runner   queryRunner
    database string
}
```

**Constructor:**

```go
func NewDriver(db neo4j.Driver, database string) *Driver
```

Creates a `Driver` wrapping a real `neo4j.Driver` via `neo4jRunner`.

**Helper:**

```go
func validateArgs(args, v any) (map[string]any, *Response, error)
```

Type-asserts `args` to `map[string]any` and `v` to `*Response`, returning descriptive errors on mismatch.

**Interface contract:**

- `Exec(ctx, query string, args, v any) error` — executes a write Cypher statement.
  - `args`: `map[string]any` (Cypher parameters).
  - `v`: `*Response` (populated with result records).
  - Uses `session.ExecuteWrite()` internally via `queryRunner.executeWrite`.
- `Query(ctx, query string, args, v any) error` — executes a read Cypher statement.
  - Same arg/result types as Exec.
  - Uses `session.ExecuteRead()` internally via `queryRunner.executeRead`.
- `Tx(ctx) (dialect.Tx, error)` — returns `dialect.NopTx(d)`. Real transactions deferred.
- `Close() error` — calls `d.runner.close(ctx)`. Returns error if runner is nil.
- `Dialect() string` — returns `dialect.Neo4j`.

```go
var _ dialect.Driver = (*Driver)(nil) // compile-time check
```

#### 3.2.2 Config (`config.go`)

```go
type Config struct {
    URI      string // bolt://host:7687, neo4j://host:7687, etc.
    Username string
    Password string
    Database string // default: "neo4j"
}

func (cfg Config) Build() (*Driver, error)
```

`Config.Build` calls `neo4j.NewDriver(cfg.URI, neo4j.BasicAuth(...))` and wraps the result with `NewDriver`. Defaults `Database` to `"neo4j"` if empty.

```go
func ParseURI(uri string) (Config, error)
```

Parses a connection URI of the form `bolt://user:pass@host:7687/dbname` into a `Config`. Strips user info from the URI, extracts credentials, and defaults database to `"neo4j"`.

#### 3.2.3 Response (`response.go`)

Wraps collected `[]*neo4j.Record` with typed read methods.

```go
type Response struct {
    records []*neo4j.Record
    columns []string
}
```

**Constructor:**

```go
func NewResponse(records []*neo4j.Record, columns []string) *Response
```

**Methods:**

- `ReadInt() (int, error)` — for `RETURN count(n)` queries. Includes bounds check on empty Values slice.
- `ReadBool() (bool, error)` — for `RETURN exists(...)` queries. Includes bounds check on empty Values slice.
- `ReadNodeMaps() ([]map[string]any, error)` — extracts node properties from all records.
- `ReadSingle() (map[string]any, error)` — extracts properties from a single record.
- `Scan(v any) error` — reads all records into `v` by building `[]map[string]any` from each record's Keys/Values pairs and decoding via JSON round-trip. Used by `group.tmpl` and `select.tmpl` for scanning aggregation and selective query results.

### 3.3 Cypher Query Builder (`dialect/neo4j/cypher/`)

Declarative clause-based builder. Assembles MATCH/WHERE/CREATE/SET/DELETE/RETURN. This type is registered as `Storage.Builder`.

#### 3.3.1 Builder Struct

```go
type Builder struct {
    match   []string
    where   []string
    create  []string
    merge   []string
    set     []string
    remove  []string
    del     []string          // named "del" to avoid Go reserved word "delete"
    ret     []string
    orderBy []string
    skip    *int
    limit   *int
    params  map[string]any
    paramN  int
}
```

#### 3.3.2 Builder Methods

```go
func New() *Builder

// Clause methods (each returns *Builder for chaining)
func (b *Builder) Match(pattern string) *Builder
func (b *Builder) Where(cond string) *Builder
func (b *Builder) Create(pattern string) *Builder
func (b *Builder) Merge(pattern string) *Builder
func (b *Builder) Set(expr string) *Builder
func (b *Builder) Remove(expr string) *Builder
func (b *Builder) Delete(expr string) *Builder
func (b *Builder) DetachDelete(expr string) *Builder   // prepends "DETACH " prefix to del entry
func (b *Builder) Return(exprs ...string) *Builder
func (b *Builder) OrderBy(expr string) *Builder
func (b *Builder) Skip(n int) *Builder
func (b *Builder) Limit(n int) *Builder

// Parameter management
func (b *Builder) AddParam(value any) string         // returns "$pN", stores in params map
func (b *Builder) SetParam(name string, value any)    // named parameter

// Predicate composition support
func (b *Builder) WhereClauses() []string              // returns raw WHERE conditions (no keyword)
func (b *Builder) Params() map[string]any              // returns the parameter map
func (b *Builder) CollectWhere(fn func(*Builder)) []string  // see below

// Output
func (b *Builder) Query() (string, map[string]any)   // returns Cypher string + params
func (b *Builder) Clone() *Builder                    // deep copy for sub-queries
```

**`CollectWhere` method:**

Applies `fn` to the builder, captures any WHERE conditions that `fn` added, removes them from the builder's where slice, and returns them. Parameters added by `fn` remain in the builder with correct sequencing. This solves two problems in predicate composition:

1. **Double WHERE**: Without `CollectWhere`, composing predicates via AND/OR/NOT would produce nested `WHERE` keywords (e.g., `WHERE WHERE n.name = $p0 AND WHERE n.age > $p1`).
2. **Param collision**: Each predicate's parameters are allocated on the same builder, so `$pN` counters remain sequential and unique.

#### 3.3.3 Predicate Functions

Generated predicates have the signature `func(*cypher.Builder)`. Each predicate appends a WHERE condition:

```go
// Example: user.NameEQ("alice") generates:
func NameEQ(v string) predicate.User {
    return func(b *cypher.Builder) {
        p := b.AddParam(v)
        b.Where(fmt.Sprintf("n.%s = %s", FieldName, p))
    }
}
```

#### 3.3.4 OpCode Mapping

| ent Op | Cypher Expression |
|---|---|
| EQ | `n.field = $p` |
| NEQ | `n.field <> $p` |
| GT | `n.field > $p` |
| GTE | `n.field >= $p` |
| LT | `n.field < $p` |
| LTE | `n.field <= $p` |
| IsNil | `n.field IS NULL` |
| NotNil | `n.field IS NOT NULL` |
| In | `n.field IN $p` |
| NotIn | `NOT n.field IN $p` |
| Contains | `n.field CONTAINS $p` |
| HasPrefix | `n.field STARTS WITH $p` |
| HasSuffix | `n.field ENDS WITH $p` |
| EqualFold | `toLower(n.field) = toLower($p)` |
| ContainsFold | `toLower(n.field) CONTAINS toLower($p)` |

**OpCode name overrides** (in `neo4jCode` array):

- `IsNil` → `"IsNull"`
- `NotNil` → `"NotNull"`
- `HasPrefix` → `"StartsWith"`
- `HasSuffix` → `"EndsWith"`

### 3.4 Node & Edge Mapping

#### 3.4.1 Nodes

- Each ent entity maps to **one Neo4j label** (e.g., `User` schema → `:User` label).
- Every node has an `id` property containing a KSUID string.
- Entity fields map 1:1 to Neo4j node properties.
- Label name = entity type name (PascalCase).

#### 3.4.2 Edges (Relationships)

- Each ent edge maps to a Neo4j relationship type.
- Relationship naming convention: `OWNER_HAS_INVERSE` (SCREAMING_SNAKE_CASE).
  - Example: `User` → `Pet` edge → `:USER_HAS_PET` relationship type.
- Direction: owner entity is the start node `(a)`, inverse entity is the end node `(b)`.
  - Pattern: `(owner)-[:OWNER_HAS_INVERSE]->(inverse)`
- Bidirectional edges use undirected pattern: `(a)-[:REL]-(b)`.

#### 3.4.3 Edge Relation Types

| ent Type | Cypher Pattern | Enforcement |
|---|---|---|
| O2O | `(a)-[:REL]->(b)` | App-level: pre-check count before create |
| O2M | `(a)-[:REL]->(b)` | Natural (one start, many ends) |
| M2O | Inverse of O2M | Traversal direction flipped in query |
| M2M | `(a)-[:REL]->(b)` | No enforcement needed |

#### 3.4.4 Uniqueness Enforcement

Application-level, not database constraints. Pattern:

```cypher
// Check for existing node with unique field value
OPTIONAL MATCH (existing:User {email: $p0})
WITH existing
WHERE existing IS NULL
CREATE (n:User {id: $p1, email: $p0, name: $p2})
RETURN n {.*}
```

If `existing IS NOT NULL`, the query returns no rows, and the driver returns a constraint error.

### 3.5 Storage Registration (`entc/gen/storage.go`)

```go
{
    Name:       "neo4j",
    IdentName:  "Neo4j",
    Builder:    reflect.TypeOf(&cypher.Builder{}),
    Dialects:   []string{"dialect.Neo4j"},
    Imports: []string{
        "entgo.io/ent/dialect/neo4j",
        "entgo.io/ent/dialect/neo4j/cypher",
    },
    SchemaMode: Unique | Indexes | Cascade,
    OpCode:     opCodes(neo4jCode[:]),
    Init:       func(*Graph) error { return nil },
}
```

**SchemaMode breakdown:**
- `Unique` — application-level unique field checks in create/update queries.
- `Indexes` — generates index annotations (for future `CREATE INDEX` support).
- `Cascade` — permanently included. All deletes use `DETACH DELETE` because Neo4j's `DELETE` fails on nodes with relationships. This matches Gremlin's `Drop()` behavior.
- `Migrate` — **excluded**. No automatic schema migration. Constraints/indexes managed externally.

### 3.6 Code Generation Templates (`entc/gen/template/dialect/neo4j/`)

13 template files mirroring the Gremlin template set. Each generates methods prefixed with `neo4j`.

#### 3.6.1 `open.tmpl` — `dialect/neo4j/client/open`

Generates the `Open(driverName, dsn)` case for Neo4j:

```go
cfg, err := neo4j.ParseURI(dataSourceName)
drv, err := cfg.Build()
return NewClient(append(options, Driver(drv))...), nil
```

#### 3.6.2 `create.tmpl` — `dialect/neo4j/create`

Generates `neo4jSave(ctx)` and `neo4j()` on CreateBuilder.

- Assigns KSUID `id` if not user-set.
- For unique fields: wraps create in `OPTIONAL MATCH` + `WITH ... WHERE existing IS NULL` guard.
- Builds `CREATE (n:Label {field: $p0, ...})`.
- For edges set during creation: appends `WITH n MATCH (m:Target) WHERE m.id = $pN CREATE (n)-[:REL]->(m)`.
- Returns `n {.*}`.

#### 3.6.3 `update.tmpl` — `dialect/neo4j/update`

Generates `neo4jSave(ctx)` for UpdateOne and bulk Update.

- `MATCH (n:Label) WHERE n.id = $p0` for UpdateOne.
- `MATCH (n:Label)` with predicate WHERE clauses for bulk Update.
- `SET n.field = $pN` for each modified field.
- `REMOVE n.field` for clearing optional fields.
- Edge mutations: `DELETE` old relationships, `CREATE` new ones in the same query using `WITH n`.

#### 3.6.4 `delete.tmpl` — `dialect/neo4j/delete`

Generates `neo4jExec(ctx)` and `neo4j()` builder method.

- Always uses `DETACH DELETE`: `MATCH (n:Label) WHERE ... DETACH DELETE n RETURN count(n)`.
- `DETACH DELETE` is always correct for Neo4j because `DELETE` fails on nodes with relationships. Since Neo4j storage permanently includes `Cascade` in SchemaMode, the conditional path in the spec is unnecessary.

#### 3.6.5 `query.tmpl` — `dialect/neo4j/query`

Generates three methods:

- `neo4jAll(ctx)` — executes query, decodes `[]T`.
- `neo4jCount(ctx)` — executes query with `RETURN count(n)`, reads int.
- `neo4jQuery(ctx) *cypher.Builder` — builds the query. If an existing builder is present (from edge traversal), clones it; otherwise creates a fresh builder with the MATCH clause. Applies predicates, ordering, and pagination.

Also defines:
- `dialect/neo4j/query/path` — edge traversal patterns: `MATCH (n:Label)-[:REL]->(m:Target)`.
- `dialect/neo4j/query/from` — reverse edge traversal for inverse edges.

#### 3.6.6 `decode.tmpl` — `dialect/neo4j/decode/one` and `dialect/neo4j/decode/many`

Generates `FromResponse(res *neo4j.Response) error` on entity types.

- Extracts `map[string]any` from Neo4j records.
- Maps property names to struct fields.
- Handles type conversions (Neo4j int64 → Go int, etc.).

#### 3.6.7 `predicate.tmpl` — Multiple sub-templates

- `dialect/neo4j/predicate/id` — `WHERE n.id = $p0`
- `dialect/neo4j/predicate/id/ops` — `WHERE n.id IN $p0`, etc.
- `dialect/neo4j/predicate/field` — `WHERE n.field = $p0`
- `dialect/neo4j/predicate/field/ops` — all OpCode operations on fields, including `EqualFold` (`toLower`) and `ContainsFold` (`toLower + CONTAINS`).
- `dialect/neo4j/predicate/edge/has` — Uses `EXISTS { ... }` subquery pattern:
  - Directed: `EXISTS { (n)-[:REL]->() }`
  - Inverse: `EXISTS { (n)<-[:REL]-() }`
  - Bidirectional: `EXISTS { (n)-[:REL]-() }`
- `dialect/neo4j/predicate/edge/haswith` — `EXISTS { ... }` with sub-builder:
  - Creates a `cypher.New()` sub-builder, applies target predicates to it, extracts the query, transfers params to the parent builder via `SetParam`, and wraps in `EXISTS { ... }`.
- `dialect/neo4j/predicate/and` — Uses `CollectWhere` to capture conditions from each predicate, joins with `AND`, wraps in parentheses.
- `dialect/neo4j/predicate/or` — Uses `CollectWhere` to capture conditions from each predicate, joins with `OR`, wraps in parentheses.
- `dialect/neo4j/predicate/not` — Uses `CollectWhere` to capture conditions, joins with `AND`, wraps in `NOT (...)`.

#### 3.6.8 `errors.tmpl` — `dialect/neo4j/errors`

Generates constraint error types. Application-level uniqueness violations return a `ConstraintError` when the guarded CREATE returns zero rows.

#### 3.6.9 `meta.tmpl` — `dialect/neo4j/meta/constants`

Generates relationship type constants:

```go
const (
    UserHasPetLabel    = "USER_HAS_PET"
    UserHasFriendLabel = "USER_HAS_FRIEND"
)
```

#### 3.6.10 `globals.tmpl` — `dialect/neo4j/globals`

Generates `type queryHook func(context.Context)` to align API surface.

#### 3.6.11 `group.tmpl` — `dialect/neo4j/group`

Generates `neo4jScan(ctx)` for GROUP BY aggregation. Builds field list from `flds`, applies aggregate functions from `fns`, constructs RETURN clause, and decodes results via `Response.Scan(v)`.

#### 3.6.12 `select.tmpl` — `dialect/neo4j/select`

Generates `neo4jScan(ctx)` for field-selective queries: `RETURN n.field1, n.field2` instead of `n {.*}`. Decodes results via `Response.Scan(v)`.

#### 3.6.13 `by.tmpl` — Multiple sub-templates

- `dialect/neo4j/order/signature` — `type OrderFunc func(*cypher.Builder)`
- `dialect/neo4j/order/func` — generates `b.OrderBy("n.field ASC")` or `DESC`
- `dialect/neo4j/group/signature` — `type AggregateFunc func(string) string`
- `dialect/neo4j/group/as`, `dialect/neo4j/group/func`, `dialect/neo4j/group/const`

## 4. Acceptance Criteria

- **Scenario**: Create a user node
  - **Given** a User entity with Neo4j storage
  - **When** `client.User.Create().SetName("alice").Save(ctx)` is called
  - **Then** a Cypher `CREATE` is executed and a `*User` is returned with a KSUID id.

- **Scenario**: Query users with predicates
  - **Given** User nodes exist in Neo4j
  - **When** `client.User.Query().Where(user.NameEQ("alice")).All(ctx)` is called
  - **Then** a Cypher `MATCH ... WHERE` query returns matching users.

- **Scenario**: Traverse edges
  - **Given** User nodes with `USER_HAS_PET` relationships to Pet nodes
  - **When** `client.User.Query().Where(user.NameEQ("alice")).QueryPets().All(ctx)` is called
  - **Then** a Cypher `MATCH (n:User)-[:USER_HAS_PET]->(m:Pet)` query returns the pets.

- **Scenario**: Delete with cascade
  - **Given** a User node with relationships
  - **When** `client.User.DeleteOneID(id).Exec(ctx)` is called
  - **Then** `DETACH DELETE` removes the node and all its relationships.

- **Scenario**: Unique field enforcement
  - **Given** a User entity with a unique `email` field
  - **When** creating a user with a duplicate email
  - **Then** a `ConstraintError` is returned.

- **Scenario**: Compose predicates with AND/OR/NOT
  - **Given** User nodes exist in Neo4j
  - **When** `client.User.Query().Where(user.And(user.NameEQ("alice"), user.AgeGT(18))).All(ctx)` is called
  - **Then** a Cypher query with `WHERE (n.name = $p0 AND n.age > $p1)` returns matching users, with no duplicate WHERE keywords and correct param sequencing.

## 5. Edge Cases

- Neo4j returns `int64` for all integers — generated decode must handle int64 → int/int32/etc. conversions.
- KSUID generation must happen before the query is built, not in the database.
- `OPTIONAL MATCH` uniqueness guard returns zero rows on violation — driver must distinguish "no rows because of constraint" from "no rows because of empty result".
- Bolt connection drops must be handled by the neo4j-go-driver's built-in retry/reconnect logic.
- Node labels are case-sensitive in Neo4j — must match entity type names exactly.
- Relationship types are case-sensitive — must use consistent SCREAMING_SNAKE_CASE.
- `nil`/zero-value fields: optional fields set to nil map to absent properties (not `null`), cleared via `REMOVE n.field`.
- Self-referential edges (e.g., User friends User): same label on both ends, relationship type still follows `OWNER_HAS_INVERSE` pattern.
- `ReadInt` / `ReadBool` must bounds-check `Values` slice to avoid index-out-of-range panics on empty records.
- `Response.Scan` uses JSON round-trip marshaling — types must be JSON-serializable and numeric precision may be affected.

## 6. Deferred Features

- **Real ACID transactions**: `Tx()` returns `NopTx` for now. Future: wrap `session.BeginTransaction()`.
- **JSON fields**: Neo4j has no native JSON type. Future: serialize to string property.
- **Relationship properties**: Neo4j supports properties on relationships. Future: extend edge mapping.
- **Database-level constraints**: `CREATE CONSTRAINT` / `CREATE INDEX` for uniqueness and indexing. Future: add a schema setup command.
- **Multiple labels per node**: Currently single label. Future: support via mixins or annotations.
