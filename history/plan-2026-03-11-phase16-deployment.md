# Phase 16: Deployment & Final Integration

## Problem Statement
Phase 16 finalizes deployment infrastructure: production Dockerfiles, Docker Compose for full stack, Fly.io configuration, Vercel config, CI/CD pipelines with coverage gates, and a Playwright E2E smoke test suite covering MVP acceptance criteria.

## Changes Made

### 1. phase16.deploy — Deployment Configuration
- **docker/Dockerfile** — Upgraded: Go 1.24, stripped binary, non-root user, healthcheck, labels
- **docker/Dockerfile.web** — Created: multi-stage Next.js build with standalone output
- **docker/docker-compose.yml** — Full stack: API + Web with proper env vars, healthchecks, networks, volumes
- **fly.toml** — Fly.io config: persistent volume, health checks, scaling, environment vars
- **web/vercel.json** — Vercel config: build settings, API rewrites, security headers
- **docs/env-vars.md** — Complete env var documentation for API, web, Docker, Fly.io, Vercel

### 2. phase16.cicd — CI/CD GitHub Actions
- **.github/workflows/ci.yml** — PR checks: Go fmt/lint/test/coverage + Next.js fmt/lint/typecheck/test/coverage + build verification
- **.github/workflows/deploy.yml** — Deploy on merge: Fly.io with health check + rollback, Vercel with verification

### 3. phase16.smoke — Playwright E2E Smoke Suite
- **web/playwright.config.ts** — Playwright configuration
- **web/e2e/helpers.ts** — Test helper functions for API interaction
- **web/e2e/smoke.spec.ts** — Comprehensive smoke tests covering all MVP acceptance criteria:
  - API Health (healthz, readyz, v1 root, RFC 7807 errors)
  - Full hierarchy lifecycle (Org → Space → Board → Thread → Message)
  - RBAC enforcement (401 on unauthenticated requests)
  - Sales pipeline flow (lead creation, metadata updates, stage transitions)
  - Billing & Voice (webhook endpoint)
  - Community features (voting, flagging)
  - Search with metadata filtering
  - Response time < 2s
  - CORS & headers (preflight, X-Request-ID)
  - Pagination
  - WebSocket endpoint
  - Notifications
