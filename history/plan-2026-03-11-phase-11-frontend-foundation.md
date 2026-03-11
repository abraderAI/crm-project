# Phase 11: Frontend Foundation Implementation Plan

## Overview
Initialize the Next.js 14+ frontend in `/web` with App Router, strict TypeScript, Tailwind CSS, shadcn/ui, Clerk auth, typed API client, theming, and base layout. All with ≥85% test coverage via Vitest + RTL.

## Tasks Completed
1. **phase11.nextjs** — Next.js 16 initialized with App Router, TypeScript, Tailwind v4, shadcn/ui utilities (cn, cva), ESLint, Prettier, Vitest with 85% coverage thresholds
2. **phase11.clerk** — Clerk auth: ClerkProvider in root layout, middleware.ts protecting all routes except sign-in/sign-up, JWT forwarded to Go API via auth().getToken()
3. **phase11.api-client** — Typed API client: TypeScript types matching all Go models, RSC serverFetch with JWT, client mutation wrapper, API error handling with ProblemDetail
4. **phase11.theming** — Theme system: CSS custom properties (13 tokens), class-based dark mode via @custom-variant, ThemeProvider with localStorage + system preference, ThemeToggle cycling light/dark/system, per-org override support
5. **phase11.layout** — Base layout: Sidebar with collapsible org/space/board nav tree, Topbar with search + notification bell + user menu + theme toggle, Breadcrumbs, responsive AppLayout shell
6. **phase11.tests** — 85 tests across 8 test files: utils, API client, ThemeProvider, ThemeToggle, Sidebar, Topbar, Breadcrumbs, AppLayout. Coverage: 97.34% stmts, 92.7% branches, 97.29% funcs, 99.03% lines

## Taskfile Updates
Added `web:*` namespaced tasks (fmt, fmt:check, lint, typecheck, test, test:coverage, build). Updated `check` task to include all web checks.
