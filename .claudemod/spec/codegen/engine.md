# Specification: Code Generation Engine

## 1. Goal

Load developer-defined schemas, build an in-memory entity graph, validate relationships, and generate type-safe Go packages with CRUD operations, query builders, and migrations.

## 2. User Stories

- **As a developer**, I want to run `go generate ./ent` to regenerate all entity code after schema changes.
- **As a developer**, I want generated code to be type-safe with compile-time guarantees.
- **As a developer**, I want to extend generation with custom templates and extensions.

## 3. Technical Requirements

- **Schema Loading** (`entc/load/`):
  - Parse Go AST of schema package.
  - Validate schema struct implementations.
  - Extract Fields, Edges, Indexes, Hooks, Interceptors, Annotations.
  - Build `SchemaSpec` from parsed definitions.

- **Graph Building** (`entc/gen/graph.go`, ~1226 lines):
  - Construct `*gen.Graph` representing all entities and their relationships.
  - Validate edge references (no dangling edges).
  - Detect cycles in required edges.
  - Assign default field values and storage keys.

- **Storage Selection** (`entc/gen/storage.go`):
  - Static array of `*Storage` structs (currently SQL and Gremlin).
  - Selected via `entc.Storage(typ)` option or CLI flag.
  - Controls which dialect-specific templates are used.
  - `SchemaMode` bitmask controls feature availability (Unique, Indexes, Cascade, Migrate).

- **Template Execution** (`entc/gen/template/`):
  - ~20 base templates for entity types, builders, client.
  - Dialect-specific overrides in `template/dialect/{sql,gremlin}/`.
  - Custom templates injectable via `entc.Extension`.
  - Output formatted with `goimports`.

- **Feature Flags** (`entc/gen/feature.go`):
  - Conditionally enable: Privacy, EntQL, GlobalID, Snapshot, SchemaConfig, Intercept.
  - Features add implicit fields/edges and enable additional templates.

## 4. Acceptance Criteria

- **Scenario**: Generate code for a User schema with SQL storage
  - **Given** a valid User schema with fields and edges
  - **When** `entc.Generate("./ent/schema")` is called
  - **Then** type-safe Go files are generated: `user.go`, `user_create.go`, `user_update.go`, `user_delete.go`, `user_query.go`, `client.go`, `migrate/`.

- **Scenario**: Add a new storage driver
  - **Given** a new `*Storage` entry in `storage.go`
  - **When** the driver is selected via config
  - **Then** dialect-specific templates are used for code generation.

## 5. Edge Cases

- Schema loading fails on invalid Go syntax.
- Circular required edges cause generation error.
- Missing edge inverse references detected and reported.
- Large schemas (100+ entities) must generate efficiently.
