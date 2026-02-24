# Specification: EntQL Query Language

## 1. Purpose

Provides a dialect-agnostic query language abstraction. EntQL predicates are translated to dialect-specific operations (SQL WHERE, Gremlin has/filter, Cypher WHERE) during code generation.

## 2. Key Components

- `entql/` — Query language types, operators, and predicate builders

## 3. Relationship to Neo4j

EntQL predicates map to Neo4j OpCodes defined in `entc/gen/storage.go`:
- `IsNil` -> `IsNull`, `NotNil` -> `NotNull`
- `HasPrefix` -> `StartsWith`, `HasSuffix` -> `EndsWith`
- Other operations use default names (EQ, NEQ, GT, LT, etc.)

## 4. Dependencies

- **Depends on:** Standard library
- **Depended on by:** `entc/gen` (predicate template generation), privacy policies
