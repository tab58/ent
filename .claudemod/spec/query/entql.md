# Specification: Query Language (EntQL)

## 1. Goal

Provide a dynamic query expression system for runtime predicate evaluation, enabling type-safe filtering across all entity types independent of the underlying dialect.

## 2. User Stories

- **As a developer**, I want to build dynamic queries at runtime using a type-safe expression API.
- **As a framework maintainer**, I want a dialect-agnostic query representation that can be translated to SQL or Gremlin.

## 3. Technical Requirements

- **Expression Types** (`entql/types.go`, ~50K lines):
  - Operations: AND, OR, NOT, EQ, NEQ, GT, GTE, LT, LTE, IN, NOT_IN.
  - Functions: HasPrefix, HasSuffix, Contains, ContainsFold, EqualFold.
  - Null checks: IsNil, NotNil.
  - Edge predicates: HasEdge, HasEdgeWith.

- **Operator System**:
  - `Op` type for comparison operators.
  - `Func` type for string/collection functions.
  - `Expr` type for composed expressions.

- **Integration**:
  - Used by privacy policies for dynamic rule evaluation.
  - Used by generated `Where()` methods for predicate composition.
  - Translated to dialect-specific predicates during query execution.

## 4. Acceptance Criteria

- **Scenario**: Dynamic predicate composition
  - **Given** an EntQL expression `And(EQ("name", "alice"), GT("age", 18))`
  - **When** translated to SQL
  - **Then** produces `name = ? AND age > ?` with parameterized values.

## 5. Edge Cases

- Deeply nested boolean expressions.
- Edge predicates across multiple relationship hops.
- Null handling in composite predicates.
