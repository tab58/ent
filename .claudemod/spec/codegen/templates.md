# Specification: Template System

## 1. Goal

Provide a Go text template system for generating dialect-specific entity code, supporting customization through template overrides and extensions.

## 2. User Stories

- **As a framework maintainer**, I want dialect-specific templates so each database backend generates appropriate query code.
- **As a developer**, I want to inject custom templates to add methods or types to generated code.
- **As an extension author**, I want to hook into the generation pipeline to add cross-cutting concerns.

## 3. Technical Requirements

- **Base Templates** (`entc/gen/template/`):
  - `base.tmpl` - Entity struct definition
  - `client.tmpl` - Client type with Open/Close
  - `context.tmpl` - Context helpers
  - `mutation.tmpl` - Mutation type
  - `where.tmpl` - Predicate functions
  - `builder/*.tmpl` - Create, Update, Delete, Query builders

- **Dialect Templates** (`entc/gen/template/dialect/{sql,gremlin}/`):
  - `create.tmpl` - Entity creation (INSERT / traversal)
  - `query.tmpl` - Entity queries (SELECT / traversal)
  - `update.tmpl` - Entity updates (UPDATE / traversal)
  - `delete.tmpl` - Entity deletion (DELETE / traversal)
  - `select.tmpl` - Select builder
  - `predicate.tmpl` - WHERE / filter predicates
  - `by.tmpl` - Ordering/grouping
  - `meta.tmpl` - Query metadata

- **Template Functions** (`entc/gen/func.go`, ~553 lines):
  - Helper functions available in templates (type conversions, naming, etc.)
  - Dialect-aware function dispatch.

- **Extension Points**:
  - `entc.Extension` can add templates, hooks, and annotations.
  - Custom templates merged into the template set before execution.
  - Template overrides replace base templates by name.

## 4. Acceptance Criteria

- **Scenario**: SQL dialect generates SQL queries
  - **Given** SQL storage is selected
  - **When** templates are executed
  - **Then** generated code uses `sql.Selector` for query building and `sqlgraph` for graph operations.

- **Scenario**: Gremlin dialect generates traversals
  - **Given** Gremlin storage is selected
  - **When** templates are executed
  - **Then** generated code uses `dsl.Traversal` for query building with GraphSON encoding.

## 5. Edge Cases

- Custom template name collisions with base templates.
- Template execution errors surface with file/line context.
- Dialect templates must handle all field types consistently.
