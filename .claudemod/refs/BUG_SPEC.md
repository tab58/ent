# ISSUE: Cross-Tenant Token Validation Bypass

## Problem Summary

In our multi-tenant SaaS platform, a user authenticated under one tenant (Org A) can make API requests that return data belonging to a different tenant (Org B) by manipulating the X-Tenant-ID header. The JWT access token encodes the user's tenant_id claim, but downstream services trust the request header instead of the token claim when resolving tenant context. This means any authenticated user can access another organization's resources by simply changing a header value.
The issue was introduced in identity-service v2.8.0 (deployed 2026-02-17) when the tenant resolution middleware was refactored to support a new onboarding flow. The previous version correctly enforced that the token's tenant_id matched the request header; the new version skips this check.

## Behaviors

### Expected

The tenantGuard middleware compares the tenant_id claim in the verified JWT against the X-Tenant-ID header on every request. If they don't match, the request is rejected with 403 Forbidden. Downstream services always receive a validated, consistent tenant context.

### Actual

The tenantGuard middleware reads tenant_id from the X-Tenant-ID header and attaches it to the request context without comparing it to the JWT claim. An authenticated user can set any X-Tenant-ID value and access resources in that tenant's scope.

## Steps to Reproduce

1. Authenticate as a user belonging to tenant org_abc123 via POST /auth/login. Note the returned JWT access token.
2. Decode the JWT and confirm the tenant_id claim is org_abc123.
3. Make a request to GET /api/v1/projects with the valid Authorization: Bearer <token> header and set X-Tenant-ID: org_xyz789 (a different tenant).
4. Observe that the response returns projects belonging to org_xyz789, not org_abc123.

## Relevant Code

**File: src/middleware/tenantGuard.ts**
This middleware runs on every authenticated route. It is responsible for establishing the tenant context used by all downstream handlers and database queries.

```typescript
// CURRENT (broken) — introduced in v2.8.0
import { Request, Response, NextFunction } from "express";
import { TenantContext } from "../types";

export function tenantGuard(req: Request, res: Response, next: NextFunction) {
  const user = req.authenticatedUser; // set by upstream JWT verification middleware

  if (!user) {
    return res.status(401).json({ error: "unauthenticated" });
  }

  // BUG: tenant_id is read from the header, not from the verified JWT claim.
  // This was changed during the onboarding refactor (PR #1187) to allow
  // "pending" users who don't yet have a tenant_id in their token to pass
  // through. The else branch was never implemented.
  const tenantId = req.headers["x-tenant-id"] as string;

  if (!tenantId) {
    return res.status(400).json({ error: "missing_tenant" });
  }

  req.tenantContext = { tenantId, userId: user.sub } as TenantContext;
  next();
}
```

**File: src/middleware/tenantGuard.ts (previous working version — v2.7.4)**

```typescript
// PREVIOUS (working)
export function tenantGuard(req: Request, res: Response, next: NextFunction) {
  const user = req.authenticatedUser;

  if (!user) {
    return res.status(401).json({ error: "unauthenticated" });
  }

  const headerTenantId = req.headers["x-tenant-id"] as string;
  const tokenTenantId = user.tenant_id; // from verified JWT claims

  if (!headerTenantId || headerTenantId !== tokenTenantId) {
    return res.status(403).json({ error: "tenant_mismatch" });
  }

  req.tenantContext = {
    tenantId: tokenTenantId,
    userId: user.sub,
  } as TenantContext;
  next();
}
```

**File: src/types.ts (relevant type, unchanged)**

```typescript
export interface TenantContext {
  tenantId: string;
  userId: string;
}

export interface AuthenticatedUser {
  sub: string; // user ID
  email: string;
  tenant_id: string; // the user's assigned tenant
  roles: string[];
  iat: number;
  exp: number;
}
```

**File: src/middleware/verifyJwt.ts (upstream middleware, unchanged, included for context)**

```typescript
import jwt from "jsonwebtoken";
import { Request, Response, NextFunction } from "express";
import { AuthenticatedUser } from "../types";

export function verifyJwt(req: Request, res: Response, next: NextFunction) {
  const token = req.headers.authorization?.replace("Bearer ", "");

  if (!token) {
    return res.status(401).json({ error: "no_token" });
  }

  try {
    const decoded = jwt.verify(token, process.env.JWT_PUBLIC_KEY!, {
      algorithms: ["RS256"],
    }) as AuthenticatedUser;

    req.authenticatedUser = decoded;
    next();
  } catch (err) {
    return res.status(401).json({ error: "invalid_token" });
  }
}
```

## Middleware Chain (request lifecycle)

```
Incoming Request
→ verifyJwt (validates token, sets req.authenticatedUser)
→ tenantGuard (should enforce tenant match, sets req.tenantContext) ← BUG IS HERE
→ route handler (uses req.tenantContext.tenantId for DB queries)
→ response
```

## Relevant Log Output

Logs from a reproduced cross-tenant request in staging (sensitive values redacted):

```
2026-02-19T10:42:03.221Z [identity-service] INFO verifyJwt: token verified
userId=usr_881a sub_tenant=org_abc123
2026-02-19T10:42:03.223Z [identity-service] INFO tenantGuard: tenant context set
tenantId=org_xyz789 userId=usr_881a
2026-02-19T10:42:03.225Z [project-service] INFO listProjects: query executed
tenant=org_xyz789 results=14
```

Note that the user usr_881a belongs to org_abc123 (per the JWT) but the tenant context is set to org_xyz789 (from the header), and the project service happily returns 14 results from the wrong tenant.

## Root Cause Analysis

PR #1187 ("Support pending users in onboarding flow") refactored tenantGuard to handle a new edge case: users who have been invited but not yet assigned to a tenant don't have a tenant_id claim in their JWT. The developer removed the comparison check intending to add a conditional branch (if user.tenant_id exists, enforce match; else, allow header-only for onboarding routes), but the conditional was never implemented. The PR passed code review without the gap being caught.

## Constraints and Requirements for the Fix

- The fix must restore the tenant_id match enforcement for all standard authenticated routes.
- The fix must still support the onboarding flow for "pending" users who lack a tenant_id in their JWT. These users should only be able to access routes under /onboarding/\*.
- The tenant_id used in req.tenantContext must always come from the verified JWT claim for users who have one — never from the request header alone.
- Existing integration tests in tests/middleware/tenantGuard.test.ts must continue to pass, and new tests must cover the cross-tenant rejection case and the pending-user onboarding case.
- The fix should not require changes to the JWT schema or the verifyJwt middleware.

## Existing Test File (for reference)

**File: tests/middleware/tenantGuard.test.ts**

```typescript
import { tenantGuard } from "../../src/middleware/tenantGuard";
import { mockRequest, mockResponse } from "../helpers";

describe("tenantGuard", () => {
  it("returns 401 if no authenticated user", () => {
    const req = mockRequest({ authenticatedUser: null });
    const res = mockResponse();
    const next = jest.fn();

    tenantGuard(req, res, next);

    expect(res.status).toHaveBeenCalledWith(401);
    expect(next).not.toHaveBeenCalled();
  });

  it("returns 400 if X-Tenant-ID header is missing", () => {
    const req = mockRequest({
      authenticatedUser: { sub: "usr_1", tenant_id: "org_abc" },
      headers: {},
    });
    const res = mockResponse();
    const next = jest.fn();

    tenantGuard(req, res, next);

    expect(res.status).toHaveBeenCalledWith(400);
    expect(next).not.toHaveBeenCalled();
  });

  // THIS TEST IS MISSING — it existed in v2.7.4 and was removed in PR #1187
  // it("returns 403 if X-Tenant-ID does not match token tenant_id")

  it("sets tenant context and calls next on valid request", () => {
    const req = mockRequest({
      authenticatedUser: { sub: "usr_1", tenant_id: "org_abc" },
      headers: { "x-tenant-id": "org_abc" },
    });
    const res = mockResponse();
    const next = jest.fn();

    tenantGuard(req, res, next);

    expect(req.tenantContext.tenantId).toBe("org_abc");
    expect(next).toHaveBeenCalled();
  });
});
```

## Acceptance Criteria

1. A request where the X-Tenant-ID header differs from the JWT's tenant\*id claim returns 403 Forbidden with { "error": "tenant_mismatch" }.
2. A request where both values match continues to work exactly as before.
3. A pending user (no tenant_id in JWT) can still access /onboarding/\* routes using the X-Tenant-ID header.
4. A pending user cannot access any route outside /onboarding/\_.
5. The req.tenantContext.tenantId is always sourced from the JWT claim for non-pending users.
6. New unit tests cover cases 1, 3, and 4. The removed regression test for cross-tenant rejection is restored.
7. No changes to verifyJwt.ts or the JWT token schema.
