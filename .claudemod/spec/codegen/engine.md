# Specification: Code Generation Engine

## 1. Purpose

The core code generation pipeline that transforms user-defined Go schemas into a fully typed client. Parses schemas via AST, builds an in-memory graph model, registers storage drivers, executes dialect-specific templates, and writes formatted Go source files.

## 2. Key Components

- `entc/entc.go` — Public API: `Generate(schemaPath, cfg, opts...)`, `LoadGraph(schemaPath, cfg)`. Extension interface for third-party hooks, templates, and annotations.
- `entc/load/` — Schema parser using `golang.org/x/tools/go/packages`. Produces `[]load.Schema` from user's schema package.
- `entc/gen/graph.go` — `Graph` type: top-level container holding `Type` nodes, `Edge` connections, and generation config. `NewGraph(cfg, schemas...)` builds the graph; `Graph.Gen()` executes templates.
- `entc/gen/type.go` — `Type` represents an entity with `Field`s, `Edge`s, `Index`es. Contains dialect-specific methods.
- `entc/gen/storage.go` — `Storage` driver registration. Maps dialect name to builder type, imports, schema capabilities, and operation codes.
- `entc/gen/template.go` — Template loading, parsing, and execution. Manages shared + dialect-specific template sets.
- `entc/gen/func.go` — Template function map (`funcmap`): type converters, string utils, Go AST helpers.

## 3. Data Models

- **Graph** — `{Nodes []*Type, Schemas []*load.Schema, Storage *Storage, Config *Config, Features []Feature}` — Top-level generation context.
- **Type** — `{Name, Fields []*Field, Edges []*Edge, Indexes []*Index, Annotations map[string]any}` — Single entity type (e.g., User, Pet).
- **Field** — `{Name, Type *field.TypeInfo, StorageKey, Validators, Default, Optional, Nillable, Immutable}` — Schema field with type info and constraints.
- **Edge** — `{Name, Type *Type, Rel Rel, Inverse bool, Unique bool, Through *Type}` — Relationship between types (O2O, O2M, M2O, M2M).
- **Storage** — `{Name, IdentName, Builder reflect.Type, Dialects []string, Imports []string, SchemaMode, OpCode func(Op) string, Init func(*Graph) error}` — Dialect driver configuration.

## 4. Interfaces

- **Public API (`entc/entc.go`):**
  - `Generate(schemaPath string, cfg *gen.Config, options ...Option) error`
  - `LoadGraph(schemaPath string, cfg *gen.Config) (*gen.Graph, error)`
- **Extension Interface:**
  - `Hooks() []gen.Hook` — Pre/post generation hooks
  - `Templates() []*gen.Template` — Additional templates
  - `Annotations() []schema.Annotation` — Metadata injected into graph
- **Storage Registration (`entc/gen/storage.go`):**
  - `NewStorage(name string) (*Storage, error)` — Lookup by name
  - `Storage.SchemaMode.Support(mode)` — Feature capability check
  - `Storage.OpCode(op) string` — Dialect-specific operation name
- **Template Execution:**
  - `Graph.Gen()` — Execute all templates, write output files
  - Templates call `{{ template "dialect/<name>/<op>" . }}` for dialect dispatch

## 5. Dependencies

- **Depends on:** `golang.org/x/tools/go/packages` (schema parsing), `text/template` (code generation), `golang.org/x/tools/imports` (goimports formatting)
- **Depended on by:** CLI tools (`cmd/ent`), all dialect packages (via storage registration), Extensions

## 6. Acceptance Criteria

- `NewGraph` builds correct `Type` nodes from `load.Schema` with all fields, edges, and indexes resolved
- Storage driver lookup by name returns correct `*Storage` for "sql", "gremlin", "neo4j"
- Neo4j storage registers: `Builder=*cypher.Builder`, `SchemaMode=Unique|Indexes|Cascade`, imports include `neo4j` and `neo4j/cypher`
- Template execution produces valid Go source for each dialect
- Generated files pass `goimports` formatting
- Extension hooks execute at correct lifecycle points

## 7. Edge Cases

- Unknown storage driver name -> `"entc/gen: invalid storage driver"` error
- Schema with circular edge references -> resolved during graph building
- Schema with no fields (edges only) -> valid generated code
- SQL-specific features (Migrate) not available in Neo4j SchemaMode

## 8. Neo4j-Specific Storage Configuration

```go
Storage{
  Name:       "neo4j",
  IdentName:  "Neo4j",
  Builder:    reflect.TypeOf(&cypher.Builder{}),
  Dialects:   []string{"dialect.Neo4j"},
  Imports:    ["entgo.io/ent/dialect/neo4j", "entgo.io/ent/dialect/neo4j/cypher"],
  SchemaMode: Unique | Indexes | Cascade,  // No Migrate
  OpCode:     {IsNil: "IsNull", NotNil: "NotNull", HasPrefix: "StartsWith", HasSuffix: "EndsWith"},
}
```
