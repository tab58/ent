# Specification: CLI Tools

## 1. Goal

Provide command-line tools for schema initialization, code generation, and schema maintenance.

## 2. User Stories

- **As a developer**, I want to initialize a new ent project with `ent init`.
- **As a developer**, I want to generate entity code with `ent generate`.
- **As a developer**, I want to fix/migrate schemas with `entfix`.

## 3. Technical Requirements

- **ent CLI** (`cmd/ent/`):
  - `ent init <Name>` - Create new schema file.
  - `ent generate ./ent/schema` - Run code generation.
  - `ent describe ./ent/schema` - Show schema information.
  - Built with Cobra CLI framework.

- **entc CLI** (`cmd/entc/`):
  - Standalone code generation executable.
  - Same generation engine as `ent generate`.

- **entfix CLI** (`cmd/entfix/`):
  - Schema migration/fixing utility.
  - Handles schema format updates between ent versions.

- **Internal** (`cmd/internal/`):
  - Shared base commands.
  - Printer utilities for formatted output.

## 4. Acceptance Criteria

- **Scenario**: Initialize a new schema
  - **Given** a developer runs `ent init User`
  - **When** the command completes
  - **Then** a `User` schema file is created in `ent/schema/user.go`.

## 5. Edge Cases

- Running generate with no schemas present.
- Conflicting schema names.
- Invalid Go syntax in schema files detected at generation time.
