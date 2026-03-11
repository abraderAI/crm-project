# Phase 1: Foundation & Infrastructure

## Problem
Establish monorepo, Go module, Docker Compose, Taskfile, CI skeleton, config system, HTTP server with Chi, and core helpers for the DEFT Evolution CRM.

## Implemented Tasks

### phase1.repo — Monorepo structure
Created `/api/cmd/server/`, `/api/internal/{domain}/`, `/api/config/`, `/api/pkg/`, `/web/`, `/docker/`, `/docs/`

### phase1.gomod — Go module + dependencies
`github.com/abraderAI/crm-project/api` with chi, gorm, modernc sqlite (glebarez/sqlite), testify, google/uuid, yaml.v3

### phase1.docker — Docker Compose
`docker/Dockerfile` (multi-stage build) + `docker/docker-compose.yml` (API + SQLite volume + web placeholder)

### phase1.taskfile — Taskfile
Root `Taskfile.yml` with: `build`, `test`, `test:coverage`, `test:fuzz`, `lint`, `fmt`, `check`, `dev`, `clean`

### phase1.ci — GitHub Actions CI
`.github/workflows/ci.yml`: run `task check` on push/PR

### phase1.config — Config loading
`api/internal/config/config.go`: env-based config with validation

### phase1.rbac-policy — RBAC policy YAML + loader
`api/config/rbac-policy.yaml` + `api/internal/config/rbac-policy.go`

### phase1.router — Chi router + middleware
`api/internal/middleware/` (request-id, logging, recovery, CORS, content-type) + `api/internal/server/server.go`

### phase1.errors — RFC 7807 error helpers
`api/pkg/errors/errors.go`: ProblemDetail struct with all error helpers

### phase1.pagination — Cursor pagination
`api/pkg/pagination/pagination.go`: UUIDv7 cursor encode/decode, parse params

### phase1.health — Health endpoints
`api/internal/health/health.go`: GET /healthz + GET /readyz with SQLite connectivity

### phase1.tests — Comprehensive tests
- Unit tests: config, RBAC policy, errors, pagination, middleware, health, database, server
- Fuzz tests: ≥50 seeds for cursor decode, env int parsing, env bool parsing
- Live API tests: real server on random port verifying health, 404→RFC 7807, CORS, request-ID, content-type enforcement
- Coverage: 97.2% (threshold: 85%)
