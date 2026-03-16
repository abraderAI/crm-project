# Plan: Admin UI Completeness & User-Facing Gaps

**Date**: 2026-03-16
**Spec**: SPECIFICATION-admin-ui-completeness.md
**vBRIEF**: vbrief/specification-admin-ui-completeness.vbrief.json
**Status**: approved

## Summary

UI gap analysis performed against all four spec modules. All 15 gaps are frontend-only —
all backend endpoints already exist. Single-phase, 13 independent tasks fully parallelisable.

## Gaps Addressed

### High Priority
1. Admin user detail page (`/admin/users/[user_id]`) — ban/unban/purge/impersonate
2. Admin system settings page (`/admin/settings`)
3. Admin security monitoring (`/admin/security`) — recent logins + failed auths
4. Self-service upgrade/conversion page (`/upgrade`)
5. User profile/account settings (`/settings`) — Clerk profile + API keys

### Medium Priority
6. Admin RBAC policy management UI (`/admin/rbac-policy`) + dry-run preview
7. Admin API usage + LLM usage stats pages
8. Admin async data exports UI (`/admin/exports`)
9. Admin platform stats KPI row on dashboard (`/admin`)
10. Chat widget preview + embed code copy in channel config
11. Phone number management UI (`/admin/channels/voice/numbers`)
12. In-app chatbot widget verification and fix

### Low Priority
13. Tier home screen naming inconsistency fix + `/reports` redirect verification

## Key Decisions

- Frontend-only — no new Go code
- Impersonation token in sessionStorage only, never localStorage
- Phone number purchase requires confirmation modal (billable action)
- All 13 tasks fully independent — maximum parallelism
- ≥85% coverage gate enforced
