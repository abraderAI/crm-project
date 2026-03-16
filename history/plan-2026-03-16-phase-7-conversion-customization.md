# Phase 7: Conversion Flows & Home Screen Customization

## Implemented

### Subphase 7.1: Tier 2 → Tier 3 Conversion

**Task 7.1.1 — Self-service upgrade flow (stub)**
- `POST /v1/me/upgrade` endpoint
- Creates org, adds user as owner, updates lead status to "converted"
- Returns new org + tier result; auth required

**Task 7.1.2 — Sales-assisted conversion**
- `POST /v1/leads/{lead_id}/convert` endpoint
- DEFT org membership required (checked in handler)
- Converts lead, creates org, promotes linked user to owner

**Task 7.1.3 — Admin override**
- `POST /v1/admin/users/{user_id}/promote` endpoint
- Platform admin only (under /admin route group)
- Audit-logged; creates org + membership

### Subphase 7.2: Home Screen Customization Polish

**Task 7.2.1 — DnD reorder + keyboard controls**
- Native HTML drag-and-drop with GripVertical handles
- `role="listbox"` + `role="option"` + `aria-grabbed` + `aria-selected`
- Arrow key navigation + Alt+Arrow for reorder + Space/Enter for toggle

**Task 7.2.2 — Reset confirmation dialog**
- `ResetConfirmDialog` component with ARIA modal
- Clicking "Reset to defaults" opens dialog; confirm executes reset

**Task 7.2.3 — E2E persistence tests**
- Tests for tiers 2, 3, 4, 5, 6 layout persistence across re-renders
- Tests for updateLayout + re-render cycle and reset + re-render cycle

## Files Created
- `api/internal/conversion/service.go`
- `api/internal/conversion/handler.go`
- `api/internal/conversion/conversion_test.go`
- `web/src/components/home/reset-confirm-dialog.tsx`
- `web/src/components/home/reset-confirm-dialog.test.tsx`

## Files Modified
- `api/internal/server/server.go` — wired new routes
- `web/src/components/home/home-layout-editor.tsx` — DnD + keyboard + reset dialog
- `web/src/components/home/home-layout-editor.test.tsx` — new tests
- `web/src/hooks/use-home-layout.test.ts` — E2E persistence tests
