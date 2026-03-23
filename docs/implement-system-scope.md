# System Scope: Metadata and API Consistency

## Background

The platform hierarchy is `system / org / space / board / thread / message`. The existing
metadata plan (`docs/implement-metadata.md`) covers org through message. This document
covers the system level and the changes required to make the full hierarchy consistent.

## Current State of the System Level

The system level already exists implicitly in the codebase. Its components are:

- **`SystemSetting`** — key-value config store; values are JSON strings; deep-merges via
  `pkg/metadata`; restricted to a `knownSettingKeys()` allowlist of 6 keys.
- **`FeatureFlag`** — feature toggles; supports an optional `OrgScope` for per-org overrides.
- **`PlatformAdmin`** — platform-wide admin user records.
- **Global spaces** — `global-support` and `global-leads`; system-owned spaces with no org.
- **`PlatformStats`** — computed read-only platform metrics.

The admin route prefix (`/v1/admin/`) already serves as the de facto system-scope designator.

---

## Q1: What Should Change to Use Metadata at System Scope?

### Finding: `SystemSetting` IS system metadata in a different shape

`GetAllSettings()` returns `{ "key": <json_value>, ... }` — a JSON object.
`UpdateSettings()` already uses `metadata.DeepMerge` under the hood.
The response shape is semantically identical to `{ "metadata": { ... } }`.

The storage shape (multi-row keyed vs. single JSON blob per entity) is the only
meaningful difference from the rest of the hierarchy.

### Change 1: Rename settings to metadata at the system level

- `GET /admin/settings` → `GET /admin/metadata`
- `PATCH /admin/settings` → `PATCH /admin/metadata`

Response shape changes from:
```json
{ "default_pipeline_stages": [...], "file_upload_limits": {...} }
```
to:
```json
{ "metadata": { "default_pipeline_stages": [...], "file_upload_limits": {...} } }
```

The known-key validation stays as a service-layer concern — it does not change the
interface shape. The 6 existing keys become validated keys *within* the metadata object.

Keep the existing `/admin/settings` path as a deprecated alias during any transition period.

**Files:** `api/internal/admin/settings.go`, `api/internal/server/server.go`

### Change 2: Fix the `rbac_policy_override` leak (prerequisite for Change 1)

`UpdateRBACOverride()` writes directly to the `system_settings` table using the key
`rbac_policy_override` — a key that is **not** in `knownSettingKeys()`.

`GetAllSettings()` performs a bare `Find` with no key filter, so it returns the RBAC
override row to any caller of `GET /admin/settings`. This is a data leak and a clear
sign the table is already being used as a general-purpose system bag beyond its stated schema.

**Required fix (choose one):**

**Option A (recommended):** Move RBAC overrides to a dedicated `rbac_policy_overrides`
table. `UpdateRBACOverride` and `GetRBACOverride` target the new table. No other changes.

**Option B:** Add `rbac_policy_override` to `knownSettingKeys()` explicitly, making it
an acknowledged system metadata key. Add a filter to `GetAllSettings` so it excludes
internal-only keys from the public settings response.

Either option must be completed before renaming `settings` → `metadata`, otherwise the
metadata endpoint would expose the RBAC override to settings readers and vice versa.

**Files:** `api/internal/admin/rbac-override.go`, `api/internal/admin/settings.go`,
`api/internal/models/system-setting.go` (if new table is added)

### Change 3: Add metadata for global spaces themselves

Global spaces (`global-support`, `global-leads`) are system-level entities, but there is
no metadata endpoint for the space itself — only for threads within it. Add:

```
GET   /admin/spaces/{space}/metadata
PATCH /admin/spaces/{space}/metadata
```

This requires the `globalspace` repository to expose `LoadMetadata` / `SetMetadata`
targeted at the `Space` record (not thread), scoped by slug. These routes are admin-only.

**Files:** `api/internal/globalspace/repository.go`, `api/internal/server/server.go`

---

## Q2: What Else Should Change to Use System Scope (Non-Metadata)?

### Finding A: `FeatureFlag.OrgScope` is a cross-boundary inconsistency

A `FeatureFlag` with `OrgScope != nil` is org-level configuration stored at the system
level. A caller cannot discover which flags apply to their org without hitting a
platform-admin-only endpoint.

**Near-term:** No structural change required. Document this as a known inconsistency.

**Future path:** Org-scoped feature flag overrides could migrate to `Org.Metadata` under
a reserved key (e.g., `feature_overrides`). This would make org-level flag state visible
at the org metadata endpoint without requiring admin access. This is a larger migration
and should be treated as a separate work item.

### Finding B: Global space URLs have no system-scope owner in the hierarchy

The route `/global-spaces/{space}/threads` is correctly isolated but the global space
itself has no owning entity that expresses the system/space relationship clearly. After
Change 3 above, `GET/PATCH /admin/spaces/{space}/metadata` covers this gap.

No further structural change is needed to the thread-level routing.

### Finding C: `PlatformStats`, `PlatformAdmin`, audit-log, and reporting routes are fine

These are read-only system-scope concerns. They do not carry arbitrary metadata and do
not cross scope boundaries. No changes required.

---

## Summary of Changes

**Prerequisite (must happen first):**
- Fix `rbac_policy_override` storage (Option A or B above)

**Metadata changes:**
- Rename `GET/PATCH /admin/settings` to `GET/PATCH /admin/metadata`; conform response shape
- Add `GET/PATCH /admin/spaces/{space}/metadata` for global-space-level metadata
- Add `LoadMetadata` / `SetMetadata` to `globalspace` repository targeting the `Space` record

**CRUD gap-fills (non-metadata, from the base metadata plan):**
- `message/service.go`: add `Metadata *string` to `UpdateInput`
- `globalspace/service.go`: add `Metadata string` to `CreateInput`

**Documentation:**
- Note `FeatureFlag.OrgScope` as a future candidate for org metadata migration
- Deprecation notice on `/admin/settings` if kept as alias

## Files to Change

**Modified:**
- `api/internal/admin/settings.go` — rename service methods; conform response shape
- `api/internal/admin/rbac-override.go` — fix storage target
- `api/internal/models/system-setting.go` — add filter or new table (per option chosen)
- `api/internal/globalspace/repository.go` — add space-level `LoadMetadata` / `SetMetadata`
- `api/internal/server/server.go` — rewire routes

**Possibly new:**
- `api/internal/models/rbac-policy-override.go` — if Option A chosen for the RBAC fix

## Dependency on Base Metadata Plan

This plan depends on `docs/implement-metadata.md`. Specifically:
- The `pkg/metadata/handler.go` shared handler must exist before system-scope metadata
  routes can be wired.
- Implement the base metadata plan first, then apply this plan on top.
