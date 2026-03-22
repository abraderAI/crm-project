# Ticket Assignment — Assignee Picker + Notification

**Date**: 2026-03-22
**Branch**: `feat/ticket-assignment`

## Summary

Added ticket assignment workflow for support tickets. DEFT org members can be assigned to tickets via a dropdown picker in the ticket detail sidebar. Assigned members receive in-app notifications via the existing event bus.

## Changes

### Backend
- Extended `globalspace.UpdateInput` with `AssignedTo` field
- `UpdateThread` validates assignee is a DEFT org member, deep-merges `assigned_to` into metadata, includes `assigned_to` in eventbus payload for notification routing
- New `GET /v1/support/deft-members` endpoint returns DEFT org members with display names
- New `ListDeftMembers` repository method + service delegation + handler (DEFT-only access)
- Route wired in `server.go`

### Frontend
- Added `DeftMember` type and `fetchDeftMembers()` API client function
- Added `assigned_to` to `UpdateSupportTicketValues`
- Assignee picker dropdown in ticket detail sidebar (DEFT-only)
- Assignee displayed in ticket info section
- System event entry auto-created on assignment change

### Tests
- Backend: table-driven tests for assignment validation, unassignment, event payload, DEFT members endpoint (repo + handler)
- Frontend: `fetchDeftMembers` API client tests
- `task check` passes: 0 lint issues, 85% Go coverage, all frontend tests pass
