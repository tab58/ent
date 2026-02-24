# Specification: Core Schema

## 1. Purpose

Provides the building blocks for defining entity schemas in Go. Users compose schemas using Fields, Edges, Indexes, and Mixins. The codegen engine parses these definitions to build the generation graph.

## 2. Key Components

- `ent.go` — `Schema` interface (`Fields`, `Edges`, `Mixin`, `Policy`, `Hooks`, `Annotations`)
- `schema/field/` — Field type definitions (string, int, float, bool, time, enum, JSON, UUID, etc.)
- `schema/edge/` — Edge/relationship definitions (To, From with O2O, O2M, M2O, M2M cardinality)
- `schema/index/` — Index definitions (single-field, composite, unique)
- `schema/mixin/` — Reusable schema fragments (e.g., TimeMixin with created_at/updated_at)

## 3. Key Types

- **field.TypeInfo** — Go type, storage type, validators, defaults, optional/nillable flags
- **edge.Descriptor** — Target type, relationship type, inverse info, unique flag, through type
- **index.Descriptor** — Field names, unique flag, storage annotations

## 4. Dependencies

- **Depends on:** Standard library
- **Depended on by:** `entc/load` (AST parsing), `entc/gen` (Graph building), user schema packages
