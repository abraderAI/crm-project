# Product Requirements Document: Deft Evolution

**Generated**: 2026-03-10
**Status**: Ready for AI Interview
**Specification Output**: /Users/johnmcdaniel/Documents/WarpDev/crm-project/SPECIFICATION.md

## Initial Input

**Project Description**: # DEFT Evolution  
Requirements Specification (Based on STRONG Architecture)

## 1. Introduction

### 1.1 Purpose
This document defines the functional and non-functional requirements for **DEFT Evolution**, the unified platform for a company selling AI agentic development tools, including on-premise solutions and hosted services versions. DEFT Evolution serves as both the pre-customer sales CRM and the post-conversion community & collaboration portal for developers and customers.

The system targets B2B technical sales cycles (lead qualification, nurturing, demos, trials, conversion) and extends into developer community engagement (forums, wikis, tickets, feature requests, product evolution discussions).

The entire platform is built around a core hierarchical threaded content model (Org → Space → Board → Thread → Message) with metadata and granular RBAC at every level. All CRM functionality (sales pipeline, lead management, billing, voice support) will be implemented **as applications running on top of this foundation** — i.e., as specialized Spaces, Boards, Threads, and custom metadata schemas.

The system will be developed rapidly using Warp (https://www.warp.dev), leveraging its AI agents for code generation, refactoring, debugging, testing, and deployment.

The foundational component — the **Spaces API** — will be built first. Every other module (sales CRM, billing, voice integration, etc.) will then be layered on top using the same data model, API structure, authentication, and permission system.

The platform is based on the STRONG architecture (strongcode.org):  
- **S**QLite (primary database)  
- **T**ypeScript  
- **R**eact  
- **O**pinionated **N**ext.js (frontend)  
- **G**o (backend services/APIs/CLIs)

This stack ensures simplicity, minimal dependencies, high performance, and ease of maintenance.

### 1.2 Scope

In Scope  
- Core hierarchical content & collaboration platform (Org → Space → Board → Thread → Message)  
- Clerk-based authentication & session management  
- Granular, inheritance-aware RBAC across all levels  
- RESTful Spaces API (v1) with metadata filtering, search, uploads, webhooks  
- **Sales CRM application** built as a set of specialized Spaces/Boards (lead management, pipeline, opportunities)  
- Lead-to-customer transition logic (converting sales records → provisioning customer Org/Space)  
- Billing Module (FlexPoint integration) applied to customer Orgs  
- Voice Support Module (Victor AI agent) with CRM logging back to customer Threads  
- Community portal features for developer collaboration (forums, wikis, tickets, feature requests)  

Out of Scope  
- Advanced multi-region replication or sharding (SQLite single-file model for MVP)  
- Custom LLM fine-tuning  
- On-premise hardware provisioning automation  
- Outbound calling/dialer  

Assumptions  
- Development uses Warp agents extensively for rapid iteration  
- Final runtime stack: Next.js (frontend + Clerk auth), Go (API/services), SQLite (data)  
- Clerk handles primary user authentication; Go backend enforces RBAC  
- Initial scale fits comfortably in SQLite; migration path to PostgreSQL preserved  
- All CRM entities (leads, opportunities, invoices, support interactions) are modeled as Threads with domain-specific metadata  

### 1.3 Overview
DEFT Evolution is a single platform where:

- Pre-sales teams use dedicated Spaces/Boards to manage leads, pipelines, and opportunities  
- Converted customers receive their own Org (or join an existing one) with Spaces for support, feature requests, documentation, etc.  
- Developers and customers collaborate in community Spaces using the same threaded model  
- All interactions share the same authentication, permissions, metadata, search, and activity timeline  

The **Spaces API** is the foundational layer — built first — providing the generic hierarchical model and access control. All domain-specific behavior (sales, billing, support) is implemented via metadata conventions, custom board rules, and UI views layered on top.

## 2. Foundational Layer – Spaces API (Built First)

### 2.1 Authentication
- Clerk for user signup, login, social auth, sessions, JWT tokens  
- Next.js middleware integrates Clerk for frontend protection  
- Go backend validates JWTs and extracts user_id for RBAC checks  

### 2.2 Hierarchy
- **Org** – tenant/company/account boundary  
- **Space** – logical grouping (Sales CRM, Customer Support, Knowledge Base, Feature Requests…)  
- **Board** – container/category (crm-leads, crm-deals, support-tickets, wiki-pages…)  
- **Thread** – primary record (lead, opportunity, ticket, wiki page, forum topic)  
- **Message** – timeline entry (note, email, call log, comment, update)  

### 2.3 RBAC & Permissions
- Roles: owner, admin, moderator, contributor, commenter, viewer  
- Membership tables: org_memberships, space_memberships, board_memberships  
- Inheritance: org → space → board (more restrictive wins)  
- Permission middleware in Go enforces access at every endpoint  

### 2.4 API (RESTful v1)
Base path: `/v1`

Core endpoints (examples):
- `/orgs` – list/create/get/patch orgs  
- `/orgs/{org}/spaces` – list/create/get/patch spaces  
- `/orgs/{org}/spaces/{space}/boards` – list/create/get/patch boards  
- `/orgs/{org}/spaces/{space}/boards/{board}/threads` – list/create/get/patch threads + metadata filters  
- `/orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/messages` – list/create messages  
- `/v1/search` – cross-level search  
- `/v1/uploads` – file attachments  
- Webhooks per org/space/board  

Metadata: JSONB at every level; deep-merge PATCH; rich filtering (`?metadata[status]=open&metadata[priority][gt]=3`)

### 2.5 Technical Foundation
- Go + GORM + SQLite (JSONB columns, indexes on common metadata keys)  
- Next.js App Router + Clerk + React Server Components for UI  
- UUIDs for all IDs; slugs for human-readable URLs  

## 3. Application Layers (Built on Top of Spaces API)

### 3.1 Sales CRM Application
- Dedicated Space: “Sales CRM” (or per-team Spaces)  
- Boards: Leads, Opportunities, Demos, Proposals  
- Threads: Individual leads/opportunities with metadata (stage, probability, value, tech_stack, assigned_to…)  
- Messages: Activity timeline (calls, emails, notes)  
- Custom views: Kanban pipeline, lead scoring dashboard  
- AI features (scoring, personalization) read/write metadata via API  

### 3.2 Lead-to-Customer Transition
- On Closed-Won:  
  - Update opportunity thread metadata  
  - Provision customer Org (or add to existing)  
  - Create default customer Spaces/Boards (Support, Feature Requests…)  
  - Link CRM user_id to Clerk/Clerk org via API  

### 3.3 Billing Module (FlexPoint)
- Applied to customer Orgs  
- Metadata on Org/Threads: billing_tier, invoices, payment_status  
- Go service handles FlexPoint API calls, updates metadata  
- UI views show billing dashboard within customer Org  

### 3.4 Voice Support Module (Victor)
- Inbound calls linked to customer Org via phone lookup  
- Conversation summaries / transcripts logged as Messages on support Threads  
- Escalation: create or update Thread with metadata (escalated_to_human, priority)  
- Warm handoff context pulled from Thread history  

### 3.5 Community / Developer Portal
- Public/community Org or per-customer Orgs  
- Spaces: Feature Requests, Discussions, Knowledge Base, Bug Reports  
- Boards: Ideas, Bugs, Documentation  
- Threads/Messages: rich markdown, attachments, voting metadata, status workflows  

## 4. Non-Functional Requirements

### 4.1 Performance & Scalability
- API/UI < 2s response (SQLite + Go)  
- Initial scale: thousands of Orgs, tens of thousands of Threads  
- Future: read replicas or PostgreSQL migration path  

### 4.2 Security & Compliance
- Clerk JWT + Go RBAC enforcement  
- PII encrypted at rest/transit  
- GDPR/CCPA readiness (export/delete per user/org)  
- PCI via FlexPoint (no card data in platform)  

### 4.3 Usability
- Responsive Next.js UI (desktop-first, mobile-capable)  
- Unified look across sales + community views  
- Dark/light mode  

### 4.4 Development & Maintainability
- Build Spaces API first (Go + SQLite schema + core endpoints)  
- Then layer CRM & community UIs/views on top  
- Use Warp agents for rapid feature generation  
- Keep code simple, typed, dependency-light (STRONG principles)  

### 4.5 Technology Stack
- Frontend: Next.js 14+ (App Router, React Server Components, TypeScript)  
- Backend: Go (Gin/Gonic or Chi + GORM)  
- Database: SQLite (JSONB, FTS5 for search)  
- Auth: Clerk  
- Payments: FlexPoint API  
- Voice: external conversational platform (Bland.ai / Retell / Twilio + LLM)  
- Deployment: Vercel (frontend) + Fly.io / Railway (Go API)  

## 5. Development Order (MVP Roadmap)
1. Spaces API foundation (schema, auth middleware, core CRUD endpoints)  
2. Clerk integration + basic Org/Space/Board UI  
3. Sales CRM application (lead/opportunity boards + pipeline view)  
4. Lead-to-customer provisioning logic  
5. Billing metadata + FlexPoint hooks  
6. Victor voice logging & escalation integration  
7. Community portal views (feature requests, discussions)  

## 6. MVP Acceptance Criteria
- Create Org → Space → Board → Thread → Message via API/UI  
- Clerk login → see only permitted content  
- Simulate sales flow: lead Thread → opportunity Thread → Closed-Won → customer Org provisioning  
- Post-conversion: create invoice metadata, log Victor interaction as Message  
- Basic dashboard views for sales pipeline and community activity  

This restructured specification positions **DEFT Evolution** as the single platform, with the **Spaces API** as the foundational layer built first. All CRM and community functionality becomes domain-specific usage of the same unified model.


**I want to build Deft Evolution that has the following features:**

---

# Specification Generation

Agent workflow for creating project specifications via structured interview.

Legend (from RFC2119): !=MUST, ~=SHOULD, ≉=SHOULD NOT, ⊗=MUST NOT, ?=MAY.

## Input Template

```
I want to build Deft Evolution that has the following features:
1. [feature]
2. [feature]
...
N. [feature]
```

## Interview Process

- ~ Use Claude AskInterviewQuestion when available (emulate it if not available)
- ! If Input Template fields are empty: ask overview, then features, then details
- ! Ask **ONE** focused, non-trivial question per step
- ⊗ ask more than one question per step; or try to sneak-in "also" questions
- ~ Provide numbered answer options when appropriate
- ! Include "other" option for custom/unknown responses
- ! make it clear which option you feel is RECOMMENDED
- ! when you are done, append to the end of this file all questions asked and answers given.

**Question Areas:**

- ! Missing decisions (language, framework, deployment)
- ! Edge cases (errors, boundaries, failure modes)
- ! Implementation details (architecture, patterns, libraries)
- ! Requirements (performance, security, scalability)
- ! UX/constraints (users, timeline, compatibility)
- ! Tradeoffs (simplicity vs features, speed vs safety)

**Completion:**

- ! Continue until little ambiguity remains
- ! Ensure spec is comprehensive enough to implement

## Output Generation

**Specification flow:**
1. ! Write `./vbrief/specification.vbrief.json` with `status: draft`
2. ! Summarize what was decided and ask the user to review
3. ! On user approval, update `status` to `approved` in the vbrief file
4. ! Run `task spec:render` (or generate `SPECIFICATION.md` directly if task unavailable)
5. ? For add-on specs: write `./vbrief/specification-{name}.vbrief.json` → `SPECIFICATION-{name}.md`

- ⊗ Write `SPECIFICATION.md` directly — it is generated from the vbrief source
- ! follow all relevant deft guidelines
- ! use RFC2119 MUST, SHOULD, MAY, SHOULD NOT, MUST NOT wording
- ! Break into phases, subphases, tasks
- ! end of each phase/subphase must implement and run testing until it passes
- ! Mark all dependencies explicitly: "Phase 2 (depends on: Phase 1)"
- ! Design for parallel work (multiple agents)
- ⊗ Write code (specification only)

## Afterwards

- ! let user know to type "implement SPECIFICATION.md" to start implementation

**Structure:**

```markdown
# Project Name

## Overview

## Requirements

## Architecture

## Implementation Plan

### Phase 1: Foundation

#### Subphase 1.1: Setup

- Task 1.1.1: (description, dependencies, acceptance criteria)

#### Subphase 1.2: Core (depends on: 1.1)

### Phase 2: Features (depends on: Phase 1)

## Testing Strategy

## Deployment
```

## Best Practices

- ! Detailed enough to implement without guesswork
- ! Clear scope boundaries (in vs out)
- ! Include rationale for major decisions
- ~ Size tasks for 1-4 hours
- ! Minimize inter-task dependencies
- ! Define clear component interfaces

## Anti-Patterns

- ⊗ Multiple questions at once
- ⊗ Assumptions without clarifying
- ⊗ Vague requirements
- ⊗ Missing dependencies
- ⊗ Sequential tasks that could be parallel

---

## Interview Questions & Answers (2026-03-11)

1. **Go HTTP Router** → Chi (lightweight, idiomatic, net/http compatible)
2. **ORM vs Raw SQL** → GORM (as specified in PRD)
3. **SQLite Driver** → modernc.org/sqlite (pure Go, no CGo)
4. **Voice Platform** → Deferred — build Thread interface, stub provider
5. **Clerk Integration Depth** → Auth only — Orgs/memberships/RBAC in SQLite
6. **UUID Format** → UUIDv7 (time-ordered, RFC 9562)
7. **API Error Format** → RFC 7807 Problem Details for errors, direct resource for success
8. **Metadata Storage** → SQLite JSON functions + generated columns for hot paths
9. **Next.js Data Fetching** → Server Components fetch Go API directly; Client Components use thin fetch wrapper
10. **Testing Strategy** → Go: testify + httptest / Next.js: Vitest + Playwright
11. **Slug Generation** → Auto-generated from name, unique per parent, numeric suffix on collision
12. **Webhook Implementation** → Full system: event filtering, HMAC signatures, delivery dashboard, manual replay
13. **File Upload Storage** → Local filesystem behind StorageProvider interface (swap to S3/R2 later)
14. **Search Implementation** → FTS5 on all levels + combined text + metadata filtering
15. **Pagination Strategy** → Cursor-based using encoded UUIDv7 timestamps
16. **Rate Limiting** → Deferred — rely on deployment platform
17. **Deployment Configuration** → Docker Compose (local dev) + Vercel (frontend) + Fly.io (backend)
18. **Soft Delete vs Hard Delete** → Soft delete everywhere (deleted_at), GDPR hard-purge as admin operation
19. **Real-time Updates** → WebSockets (nhooyr.io/websocket) for full bidirectional communication
20. **Lead Scoring AI** → Hybrid: rule-based baseline + optional LLM enrichment (Grok default, provider-agnostic interface)
21. **FlexPoint Billing** → Full integration behind BillingProvider abstraction
22. **UI Component Library** → shadcn/ui + Tailwind CSS
23. **Monorepo vs Separate Repos** → Monorepo (/api, /web, /docker, /docs). Backend fully tested before frontend.
24. **Go Project Structure** → Flat domain-oriented under /api/internal/{domain}/
25. **RBAC Permission Resolution** → Explicit override with parent fallback (configurable via YAML policy file)
26. **WebSocket Library** → nhooyr.io/websocket (context.Context-native, fits Chi + net/http stack)
27. **Logging & Observability** → slog + OpenTelemetry (traces, metrics)
28. **CI/CD Pipeline** → GitHub Actions
29. **Lead-to-Customer Provisioning** → Fully automated on Closed-Won
30. **Dark/Light Mode** → System preference with toggle + per-org branding via CSS custom properties
31. **Markdown Rendering** → Tiptap WYSIWYG (GFM, mermaid, syntax highlight, raw toggle); stored as markdown; rendered via react-markdown
32. **Community Voting** → Weighted by role/billing tier, configurable weights
33. **Notification System** → In-app + email (Resend) + digests, behind provider abstraction (extensible to Slack/Teams/SMS)
34. **Email Provider** → Resend + React Email templates
35. **API Key Authentication** → Per-org, hashed, prefixed (deft_live_xxx), scoped permissions
36. **Audit Logging** → Full: every mutation with who/what/when/before-after diff/IP/request ID
37. **LLM Provider** → Provider-agnostic interface, Grok as default implementation
38. **SQLite Backup** → Deferred; WAL mode enabled, Litestream-ready architecture
39. **File Upload Constraints** → 100MB max; images, docs, text, archives, video, audio
40. **Thread/Message Edit History** → Full revision tracking, every edit creates revision record
41. **Default CRM Pipeline** → new_lead → qualified → demo_scheduled → demo_completed → proposal_sent → negotiation → closed_won/closed_lost + nurturing; per-org configurable
42. **Community Moderation** → Role-based: pin/lock/hide/move/merge + user flagging + moderator queue
43. **API Versioning** → URL path (/v1/, /v2/) with deprecation periods

