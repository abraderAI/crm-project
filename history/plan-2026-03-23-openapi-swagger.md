# OpenAPI 3.1 Spec + Swagger UI Implementation

**Date**: 2026-03-23
**Status**: Completed

## Summary

Added a comprehensive OpenAPI 3.1 specification and Swagger UI to the DEFT Evolution CRM REST API.

## Files Created/Modified

- `api/openapi.yaml` — Full OpenAPI 3.1 spec (~4700 lines, 100+ endpoints, 25+ schemas, 13 enums)
- `api/internal/docs/docs.go` — Go package using `//go:embed` to serve the spec and a CDN-based Swagger UI HTML page
- `api/internal/docs/openapi.yaml` — Embedded copy of the spec (synced via `task go:openapi:sync`)
- `api/internal/server/server.go` — Added `/docs` (Swagger UI) and `/docs/openapi.yaml` routes
- `Taskfile.yml` — Added `go:openapi:sync` task to copy spec into embed directory
- `vbrief/plan.vbrief.json` — Updated plan tracking

## Endpoints Documented

All 100+ endpoints across: health, orgs, spaces, boards, threads, messages, members, votes,
moderation, webhooks, notifications, uploads, revisions, search, billing, voice, channels,
chat, pipeline, admin, reporting, global-spaces, support, tier, conversion, api-keys, websocket.

## Approach

- Hand-written YAML spec (no code-gen annotations needed)
- CDN-based Swagger UI (no Go dependency, single embedded HTML page)
- `go:embed` bundles spec into binary at compile time
- Spec file at `api/openapi.yaml` is canonical; copied to `api/internal/docs/` for embedding
