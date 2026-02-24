# Specification: CLI Tools

## 1. Purpose

Command-line interface for ent code generation. Provides `init`, `generate`, and `describe` commands via cobra.

## 2. Key Components

- `cmd/ent/ent.go` — Primary CLI entry point (recommended)
- `cmd/entc/entc.go` — Legacy CLI (older, fewer features)
- `cmd/entfix/` — Migration/fix utilities for schema upgrades

## 3. Commands

- `ent init <TypeName>` — Scaffold a new schema file
- `ent generate ./ent/schema` — Run code generation
- `ent describe ./ent/schema` — Print schema description

## 4. Dependencies

- **Depends on:** `entc/entc.go` (Generate, LoadGraph), `github.com/spf13/cobra`
- **Depended on by:** User workflows, CI pipelines
