# Support Ticket Visibility Scope Enforcement

**Date**: 2026-03-21
**Branch**: `feat/support-ticket-visibility-scope`
**Status**: Completed

## Summary

Enforced server-side visibility scoping on all support ticket endpoints in `globalspace/`.
Previously, any authenticated user could list or view any support ticket — the `mine` and
`org_id` query params were optional convenience filters, not access controls.

## Three-Tier Visibility Model

- **ScopeAll** (DEFT employees / platform admins): See all tickets across all orgs.
- **ScopeOrg** (org members): See tickets belonging to their org(s).
- **ScopeOwner** (individual users, no org): See only tickets they authored or are assigned to.

## Files Changed

- `api/internal/globalspace/repository.go` — `FindUserOrgIDs`, `IsDeftOrAdmin`, `VisibleOrgIDs`/`VisibleUserID` query filters
- `api/internal/globalspace/service.go` — `VisibilityScope` type, `CallerVisibility`, `ResolveVisibility`, `canSeeThread`, enforced on all endpoints
- `api/internal/globalspace/handler.go` — `resolveVisibility` helper, auth required for support space, visibility passed to all service calls
- `api/internal/globalspace/handler_test.go` — comprehensive visibility tests for all three tiers + mutation gating
- `vbrief/plan.vbrief.json` — updated
