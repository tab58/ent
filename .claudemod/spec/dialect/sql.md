# Specification: SQL Dialect (Reference)

## 1. Purpose

Production SQL backend supporting MySQL, PostgreSQL, and SQLite. The most mature dialect with full feature support including schema migration via Atlas.

## 2. Key Components

- `dialect/sql/` — SQL driver, query builder (`Selector`, `InsertBuilder`, `UpdateBuilder`), sqlgraph traversals
- `dialect/entsql/` — SQL-specific schema annotations (table names, column types)
- `entc/gen/template/dialect/sql/` — SQL-specific code generation templates

## 3. Relevance to Neo4j

SQL serves as the reference implementation. Neo4j follows the same pattern:
- Storage registration in `storage.go` (same structure)
- Template set (same 13 template files)
- OpCode mappings (subset of SQL's)

Key difference: SQL supports `Migrate` SchemaMode (DDL generation); Neo4j does not.

## 4. Dependencies

- **Depends on:** `database/sql`, `ariga.io/atlas`
- **Depended on by:** Generated SQL client code
