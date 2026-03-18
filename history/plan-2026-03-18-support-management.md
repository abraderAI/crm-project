# Plan: Support Tickets Management Page

**Date**: 2026-03-18
**Branch**: feat/support-management-page
**Status**: Implemented

## Problem

The existing `/support` page only showed a user's own tickets with a basic create form (`SupportView`). It had no RBAC awareness, while higher-tier users need org-scoped or global views and dashboard-style visibility.

## RBAC Matrix

| Tier | Role | Scope | Stats |
|------|------|-------|-------|
| 1 | Anonymous | Sign-in prompt, no tickets | No |
| 2 | Registered developer | Own tickets (mine=true) | No |
| 3 (no org) | Paying customer | Own tickets (mine=true) | No |
| 3 (with org) | Paying customer in org | Org tickets (org_id) | Yes |
| 4 | DEFT employee (any dept) | All tickets | Yes |
| 5 (subType=owner) | Customer org admin | Org tickets (org_id) + "Organization Support" | Yes |
| 5 (DEFT dept) | DEFT support admin | All tickets + "Support Dashboard" | Yes |
| 6 | Platform admin | All tickets + "Support Dashboard" | Yes |

## Changes Made

### 1. `web/src/lib/global-api.ts`
- Added `GlobalSupportParams` interface (mine?, org_id?, limit?, cursor?)
- Added `fetchGlobalSupportTickets(token, params?)` function targeting `GET /global-spaces/global-support/threads`

### 2. `web/src/lib/global-api.test.ts`
- Added `fetchGlobalSupportTickets` describe block with 10 tests covering all param combinations, auth header, pagination, and error handling

### 3. `web/src/components/support/support-management-view.tsx` (new)
- Client component using `useTier` for RBAC scope derivation
- Scope logic: `scopesAll`, `scopesOrg`, `scopesMine` derived from tier/subType/orgId
- Stats strip (open/pending/resolved counts) for org and global scopes
- Status and search client-side filters
- Inline create ticket form with org_id passthrough
- Paginated list with load-more
- Tier-appropriate page headings

### 4. `web/src/components/support/support-management-view.test.tsx` (new)
- 70+ tests covering all tier scenarios, RBAC assertions, list rendering, status badges, filter behavior, create form (toggle/success/error/null-token), and pagination with cursor

### 5. `web/src/app/support/page.tsx`
- Replaced SSR `SupportView` wrapper with thin server component rendering `SupportManagementView`

## Files Unchanged
- `SupportView` (kept for backward compatibility)
- `app-layout-wrapper.tsx` (no new nav item needed; `/support` already present)
- `user-api.ts` (no new server-side fetch needed)
