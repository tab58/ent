# Specification: Privacy & Access Control

## 1. Goal

Provide a policy-based access control framework that evaluates rules on every query and mutation, enabling fine-grained data access control at the entity level.

## 2. User Stories

- **As a developer**, I want to define privacy policies on my entities to control who can read or write data.
- **As a developer**, I want policies to be automatically enforced on all queries without manual checks.

## 3. Technical Requirements

- **Rule Interfaces** (`privacy/privacy.go`, ~100 lines):
  - `QueryRule`: Evaluates on read operations. Returns Allow, Deny, or Skip.
  - `MutationRule`: Evaluates on write operations. Returns Allow, Deny, or Skip.
  - `QueryMutationRule`: Applies to both reads and writes.

- **Decision Model**:
  - `Allow`: Grants access, stops evaluation.
  - `Deny`: Denies access, stops evaluation.
  - `Skip`: Continues to next rule.
  - If all rules Skip, access is denied by default.

- **Integration**:
  - Policies defined on schemas via `Policy()` method.
  - Generated hooks enforce policies before query/mutation execution.
  - EntQL expressions used for dynamic rule conditions.

## 4. Acceptance Criteria

- **Scenario**: Tenant isolation policy
  - **Given** a privacy policy requiring `tenant_id` match
  - **When** a user queries data from another tenant
  - **Then** the query returns empty results (denied).

## 5. Edge Cases

- Admin bypass policies.
- Policy evaluation order matters (first Allow/Deny wins).
- Performance impact of complex privacy rules on large queries.
