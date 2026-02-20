# Specification: Core Schema Definition

## 1. Goal

Provide a Go-native API for defining database entity schemas using builder patterns, enabling type-safe schema definitions with fields, edges (relationships), indexes, mixins, and annotations.

## 2. User Stories

- **As a developer**, I want to define entity schemas as Go structs so I get IDE autocompletion and compile-time checks.
- **As a developer**, I want to define relationships between entities using edges (To/From) with cardinality support.
- **As a developer**, I want to compose schemas using mixins for shared fields and logic.
- **As a developer**, I want to attach metadata via annotations for dialect-specific behavior.

## 3. Technical Requirements

- **Core Interface** (`ent.go`): `Schema` interface with `Fields()`, `Edges()`, `Mixin()`, `Policy()`, `Hooks()`, `Interceptors()`, `Indexes()`, `Annotations()` methods.
- **Field Types** (`schema/field/`): String, Int, Float, Bool, Time, Enum, UUID, JSON, Bytes, Other with builder methods for validation, defaults, immutability, sensitivity, and comments.
- **Edge Types** (`schema/edge/`): `To()` (O2M, M2M), `From()` (inverse), with `Unique()`, `Required()`, `Comment()`, `StorageKey()`, `Through()` modifiers.
- **Index Types** (`schema/index/`): Single and composite indexes with `Unique()`, `StorageKey()`, `Annotations()`.
- **Mixin Types** (`schema/mixin/`): `Schema` embedding for shared field sets (e.g., timestamps, soft-delete).

## 4. Acceptance Criteria

- **Scenario**: Developer defines a User schema
  - **Given** a Go struct implementing `ent.Schema`
  - **When** `Fields()` returns field descriptors and `Edges()` returns edge descriptors
  - **Then** the schema is loadable by `entc/load` and generates correct entity code.

- **Scenario**: Schema uses a mixin
  - **Given** a mixin implementing `ent.Mixin`
  - **When** the schema includes it via `Mixin()`
  - **Then** mixin fields and hooks are merged into the schema.

## 5. Edge Cases

- Circular edge references must be detected and reported.
- Duplicate field names across mixins and schema must error.
- Self-referential edges (e.g., User follows User) must be supported.
- Optional vs required edge enforcement at generation time.
