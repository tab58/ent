# Specification: Gremlin Dialect (Reference)

## 1. Purpose

Graph database backend using Apache TinkerPop Gremlin traversal language. The closest architectural precedent for the Neo4j dialect.

## 2. Key Components

- `dialect/gremlin/` — Gremlin driver (HTTP/WebSocket), DSL traversal builder
- `dialect/gremlin/graph/dsl/` — Gremlin DSL: `Traversal`, `__` (anonymous traversals), `g` (graph source), `p` (predicates)
- `entc/gen/template/dialect/gremlin/` — Gremlin-specific code generation templates

## 3. Relevance to Neo4j

Neo4j was modeled after Gremlin:
- Same template definition set (verified by `TestNeo4jTemplateParity`)
- Same SchemaMode flags (Unique only for Gremlin; Unique|Indexes|Cascade for Neo4j)
- Similar graph-first data model (nodes + edges vs. tables + joins)
- ID strategy: string-based (Gremlin uses server-assigned; Neo4j uses KSUID)

## 4. Dependencies

- **Depends on:** `entgo.io/ent/dialect/gremlin/encoding/graphson`
- **Depended on by:** Generated Gremlin client code
