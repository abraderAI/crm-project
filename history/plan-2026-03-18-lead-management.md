# Lead Management Page for DEFT Sales Employees

**Date**: 2026-03-18
**Branch**: feat/lead-management-page

## Problem

DEFT sales employees had no dedicated UI to review and manage sales leads. The existing CRM Pipeline (`/crm`) shows org-scoped leads in a Kanban view for all users. A separate, access-controlled leads list page is needed for DEFT internal staff to manage leads from the `global-leads` space.

## RBAC Rules

- `tier >= 5` (org manager or platform admin): sees **all** leads
- `tier === 4 && deftDepartment === "sales"` (sales rep): sees only **own + assigned** leads (`mine=true`)
- All other tiers: access denied

## Changes Implemented

### API additions — `web/src/lib/global-api.ts`
- `fetchGlobalLeads(token, params?)` — `GET /global-spaces/global-leads/threads` with optional `mine=true`
- `fetchGlobalThread(spaceSlug, threadSlug, token?)` — `GET /global-spaces/${spaceSlug}/threads/${threadSlug}`

### Server-side API — `web/src/lib/user-api.ts`
- `fetchLeads(params?)` — server-side paginated fetch (same pattern as `fetchSupportTickets`)
- `fetchGlobalLeadThread(threadSlug)` — server-side single thread fetch for the detail page

### New component — `web/src/components/crm/leads-management-view.tsx`
Client component with tier-based RBAC, `fetchGlobalLeads`, stage/assignee/search filters, pagination.

### New routes
- `web/src/app/crm/leads/page.tsx` — index page for leads management
- `web/src/app/crm/leads/global/[thread_slug]/page.tsx` — global lead detail page

### Navigation
- `app-layout-wrapper.tsx` — added "Leads" nav item at `/crm/leads`

## Tests

- `global-api.test.ts` — 13 new tests for `fetchGlobalLeads` and `fetchGlobalThread`
- `leads-management-view.test.tsx` — 43 tests covering RBAC for all tiers, loading states, rendering, error handling, filters, and pagination
