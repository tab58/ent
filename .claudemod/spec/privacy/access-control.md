# Specification: Privacy / Access Control

## 1. Purpose

Policy-based authorization framework. Privacy policies are defined on schemas and enforced at query time via generated hooks. Supports `Allow`, `Deny`, and `Skip` decisions.

## 2. Key Components

- `privacy/` — Policy interface, decision types, rule combinators

## 3. Relationship to Neo4j

Privacy policies are dialect-agnostic. They wrap generated query/mutation methods with authorization checks. No Neo4j-specific privacy logic exists.

## 4. Dependencies

- **Depends on:** `entgo.io/ent` (Schema interface)
- **Depended on by:** Generated client code (hooks), user schema policies
