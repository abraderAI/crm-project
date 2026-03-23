# Metadata API: Simple + Comprehensive Coverage

## Problem

Metadata exists on org/space/board/thread/message models, but the REST surface is inconsistent: some endpoints allow metadata updates, others do not, and there is no single metadata-only API shape across the hierarchy.

## Design Goal

Provide one consistent metadata sub-resource pattern for every hierarchy level, while preserving existing CRUD behavior and reusing current `pkg/metadata` deep-merge semantics.

## Proposed API Shape

Add metadata sub-resources for each entity:

```
GET    /orgs/{org}/metadata
PATCH  /orgs/{org}/metadata

GET    /orgs/{org}/spaces/{space}/metadata
PATCH  /orgs/{org}/spaces/{space}/metadata

GET    /orgs/{org}/spaces/{space}/boards/{board}/metadata
PATCH  /orgs/{org}/spaces/{space}/boards/{board}/metadata

GET    /orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/metadata
PATCH  /orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/metadata

GET    /orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/messages/{message}/metadata
PATCH  /orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/messages/{message}/metadata

GET    /global-spaces/{space}/threads/{slug}/metadata
PATCH  /global-spaces/{space}/threads/{slug}/metadata
```

Request body for PATCH is a JSON object patch.
Response for GET/PATCH is `{ "metadata": { ... } }`.

## Core Semantics

Reuse existing `api/pkg/metadata/metadata.go`:

- Validate request body as a JSON object (not array/scalar)
- Deep-merge patch into stored metadata
- `null` values delete keys
- Arrays and scalars overwrite
- `{}` body is a no-op

## Implementation Strategy

### 1. Shared metadata handler (`api/pkg/metadata/handler.go`)

Define a small two-method interface:

```go
type Store interface {
    LoadMetadata(ctx context.Context, id string) (string, error)
    SetMetadata(ctx context.Context, id, merged string) error
}
```

The shared `Handler` struct takes a `Store` and a function `resolveID(r *http.Request) string` to extract the entity ID from the URL. It exposes two HTTP methods:

- `Get(w, r)` — loads and returns `{ "metadata": <parsed object> }`
- `Patch(w, r)` — reads body, validates, deep-merges, persists, returns updated metadata

All validation, merge, error handling, and response encoding lives here once.

### 2. Repository adapters per entity

Add `LoadMetadata` and `SetMetadata` methods to existing repositories using a targeted single-column update:

- `api/internal/org/repository.go`
- `api/internal/space/repository.go`
- `api/internal/board/repository.go`
- `api/internal/thread/repository.go`
- `api/internal/message/repository.go`
- `api/internal/globalspace/repository.go`

Each method resolves the entity using existing scoped lookup rules (e.g. by ID within the correct org/space/board scope) and updates only the `metadata` column.

### 3. Route wiring (`api/internal/server/server.go`)

Mount the new handler pairs under existing authenticated route groups so they inherit auth, ban, suspension, and membership middleware automatically. Example:

```go
// Existing org routes
authed.Get("/orgs/{org}/metadata",  orgMetaHandler.Get)
authed.Patch("/orgs/{org}/metadata", orgMetaHandler.Patch)

// Inside the board subrouter
bd.Get("/{board}/metadata",  boardMetaHandler.Get)
bd.Patch("/{board}/metadata", boardMetaHandler.Patch)
```

Wire via `server-services.go` where other handlers are constructed.

### 4. Fill current gaps in regular CRUD

These are separate gap-fixes independent of the sub-resource:

- **`api/internal/message/service.go`**: add `Metadata *string` to `UpdateInput`; deep-merge on update alongside existing `Body` update.
- **`api/internal/globalspace/service.go`**: add `Metadata string` to `CreateInput`; validate and default to `{}` on create.

These keep existing CRUD endpoints behaviorally complete while the new metadata sub-resource becomes the canonical metadata-only path.

## Error Model

| Condition | Status |
|---|---|
| Entity not found | 404 |
| Body is not valid JSON object | 400 |
| Merge/validation failure | 422 |
| Persistence failure | 500 |

Use existing `pkg/errors` helpers (`BadRequest`, `NotFound`, `ValidationError`, `InternalError`).

## Testing

Add focused tests covering:

- `GET` returns current metadata as a parsed JSON object
- `PATCH` deep-merges nested objects correctly
- `PATCH` with `null` values removes keys
- `PATCH` with `{}` is a no-op
- Invalid payload (non-object, malformed JSON) returns 400/422
- Message `UpdateInput` now accepts and persists metadata
- Globalspace thread create persists caller-supplied metadata

## Files to Change

**New:**
- `api/pkg/metadata/handler.go`
- `api/pkg/metadata/handler_test.go`

**Modified:**
- `api/internal/org/repository.go` — add `LoadMetadata`, `SetMetadata`
- `api/internal/space/repository.go` — add `LoadMetadata`, `SetMetadata`
- `api/internal/board/repository.go` — add `LoadMetadata`, `SetMetadata`
- `api/internal/thread/repository.go` — add `LoadMetadata`, `SetMetadata`
- `api/internal/message/repository.go` — add `LoadMetadata`, `SetMetadata`
- `api/internal/globalspace/repository.go` — add `LoadMetadata`, `SetMetadata`
- `api/internal/message/service.go` — add `Metadata *string` to `UpdateInput`
- `api/internal/globalspace/service.go` — add `Metadata string` to `CreateInput`
- `api/internal/server/server-services.go` — construct metadata handlers
- `api/internal/server/server.go` — wire new routes

## Rollout Notes

- Existing inline `metadata` fields in CRUD payloads remain unchanged for backward compatibility.
- Metadata sub-resources should be documented as the preferred metadata-only endpoints in the OpenAPI spec.
- No migrations required — `metadata` columns already exist on all five tables.
