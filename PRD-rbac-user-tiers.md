# DEFT Evolution — User Tiers & Role-Based Access PRD

## Problem Statement

The DEFT Evolution platform has no enforcement of role-based access. All authenticated
users see identical interfaces regardless of whether they are anonymous visitors, free
developers, paying customers, DEFT employees, org administrators, or platform admins.
This creates a poor user experience, exposes privileged features to the wrong audiences,
and misses lead capture and conversion opportunities.

## Goals

- Define and enforce 6 distinct user tiers, each with a tailored home screen
- Model all content (docs, forums, support tickets, leads) as tagged threads in four global spaces
- Capture anonymous visitors as leads and seamlessly promote their identity on registration
- Gate features per tier (read-only docs for anon; billing only for paying customers; CRM only for DEFT sales)
- Provide an AI-powered chatbot with live agent handoff available to all tiers
- Enable home screen widget customization (show/hide/reorder) for all authenticated tiers

## Non-Goals

- Full billing/provisioning system implementation (placeholder UI only in this phase)
- Building a dedicated documentation site (docs live as tagged threads in `global-docs`)
- LiveKit voice channel configuration (separate existing phase)
- Choice of AI model/provider for chatbot (deferred to chatbot implementation phase)

---

## User Tiers

### Tier 1 — Anonymous Visitor
A non-registered user browsing public content. No Clerk account required. A Clerk
anonymous token is issued for chatbot sessions. Should be captured and tracked as a
CRM lead.

### Tier 2 — Registered Developer (Free)
A Clerk-registered user who has not converted to a paying customer. Can participate in
the public forum, open support tickets, and access personal account features. Should be
tracked as a CRM lead until converted.

### Tier 3 — Paying Customer Org Member
A registered user belonging to a paying customer org. Can open org-scoped support
tickets and view their org's ticket history. The org **owner** also has access to the
billing/provisioning placeholder UI and an org-level support dashboard.

### Tier 4 — DEFT Employee
A member of the internal `deft` org. Home screen is department-specific based on their
space membership:
- **Sales space**: lead pipeline and conversion CRM view
- **Support space**: support ticket queue and stats
- **Finance space**: billing overview and revenue summary

### Tier 5 — Customer Org Admin
An `admin` or `owner` role member of a paying customer org. Sees all Tier 3 features
plus full org access control management (assign roles, add/remove members, manage spaces).

### Tier 6 — Platform Admin
A `platform_admin` user. Full access to all admin routes, system health, audit log,
feature flags, user management, impersonation, and all DEFT operational data.

---

## Tier Resolution

Tier is resolved server-side at login using **highest-privilege-wins**:

```
platform_admin > DEFT org member > customer org admin > customer org member > registered user > anonymous
```

Client-supplied tier claims are never trusted.

---

## Global Content Spaces

All user-generated content is modeled as threads with a `thread_type` tag within one
of four global spaces. This replaces ad-hoc org-scoped content for cross-cutting concerns.

| Space | Thread Types | Read Access | Write Access |
|---|---|---|---|
| `global-docs` | `wiki`, `documentation`, `tutorial` | Public (no auth) | DEFT org members |
| `global-forum` | `forum` | Public (no auth) | Tier 2+ |
| `global-support` | `support` | Auth-required; filtered by org | Tier 2+ (create); DEFT support (all) |
| `global-leads` | `lead` | DEFT org only | System + DEFT sales |

Support tickets in `global-support` carry an `org_id` field. Tier 3 members see only
threads where `org_id` matches their org. DEFT support space members see all threads.

---

## Chatbot

An AI-powered chatbot widget is available on all pages to all tiers including anonymous
visitors.

- **Initial interaction**: AI bot answers questions using RAG over `global-docs` content
- **Lead capture**: bot collects name/email during conversation; creates a lead record in `global-leads`
- **Escalation**: bot can hand off to a live DEFT support agent via the IO channels chat system
- **Anonymous identity**: Clerk anonymous token issued on first chatbot interaction; promoted
  to full account on registration, linking all prior sessions and lead activity

---

## Conversion Paths (Tier 2 → Tier 3)

Three paths are supported:

1. **Self-service** — user clicks "Upgrade", completes payment flow, org auto-created
2. **Sales-assisted** — DEFT sales member marks lead as converted in CRM, triggering promotion and org creation
3. **Admin override** — platform admin manually promotes user and creates org

---

## Home Screen Widgets (Default by Tier)

| Tier | Default Widgets |
|---|---|
| T1 Anonymous | Public docs highlights, forum highlights, "Get Started" CTA, chatbot |
| T2 Registered | Profile summary, recent forum activity, my support tickets, upgrade CTA, chatbot |
| T3 Org Member | Org overview, org support tickets (filtered), forum activity, chatbot |
| T3 Org Owner | Org support dashboard, billing status (placeholder), org member summary, chatbot |
| T5 Org Admin | Org access controls, member list + roles, org support dashboard, billing placeholder |
| T4 Sales | Lead pipeline, recent leads, conversion metrics |
| T4 Support | Open ticket queue, ticket stats, recent escalations |
| T4 Finance | Billing overview, revenue summary, recent payments |
| T6 Platform Admin | System health, recent audit log, user management, feature flags, all T4 widgets |

All authenticated tiers can show/hide and reorder their home screen widgets. Preferences
persist per user.

---

## Functional Requirements

**FR-1**: The system MUST resolve a user's tier server-side using highest-privilege-wins logic.

**FR-2**: Each tier MUST have a distinct default home screen with tier-appropriate widgets.

**FR-3**: Authenticated users MAY customize their home screen (show/hide/reorder widgets). Preferences MUST persist per user via the API.

**FR-4**: All content MUST be modeled as threads with `thread_type` and `visibility` fields within the four global spaces.

**FR-5**: `global-docs` MUST be publicly readable without authentication.

**FR-6**: `global-forum` MUST be publicly readable; Tier 2+ MAY create and comment on threads.

**FR-7**: `global-support` MUST require authentication; Tier 3+ org members see only their org's tickets; DEFT support space members see all tickets.

**FR-8**: `global-leads` MUST be visible only to DEFT org members.

**FR-9**: An AI chatbot widget MUST be available on all pages to all tiers including anonymous visitors.

**FR-10**: The chatbot MUST support live agent handoff to DEFT support staff via IO channels chat.

**FR-11**: Clerk anonymous sessions MUST be promoted to full accounts on registration, linking all prior chatbot sessions and lead records.

**FR-12**: Tier 2 → Tier 3 conversion MUST be supported via self-service, sales-assisted, and admin override paths.

**FR-13**: Tier 5 org admins MUST have access to org RBAC management (assign roles, manage members).

**FR-14**: Tier 4 home screens MUST default to department-specific widgets based on DEFT org space membership.

**FR-15**: Billing and provisioning UI MUST be present as a stub for Tier 3 org owners and Tier 6 admins; full implementation is deferred.

## Non-Functional Requirements

**NFR-1**: Tier resolution MUST be computed server-side; client-supplied tier claims MUST NOT be trusted.

**NFR-2**: Public routes (`global-docs`, `global-forum` reads) MUST NOT require auth or trigger redirects.

**NFR-3**: All `/admin/*` routes MUST be inaccessible to non-`platform_admin` users; access attempts MUST be audit-logged.

**NFR-4**: The chatbot widget MUST load asynchronously and not block page render.

**NFR-5**: Home screen widget preferences MUST be available at initial page render (SSR or pre-hydration API call).

---

## Success Metrics

- Anonymous visitors are captured as leads with >0 chatbot interaction rate
- Tier 2 users can successfully create forum posts and support tickets
- Tier 3 org members see only their org's tickets (no cross-org data leakage)
- Tier 4 DEFT employees see department-specific home screen by default
- Home screen widget preferences persist across sessions
- No unauthenticated access to `/admin/*` or `global-leads`

## Open Questions

- **OQ-1**: Payment processor for self-service Tier 2→3 conversion (Stripe assumed; deferred)
- **OQ-2**: AI model/provider for chatbot (deferred to chatbot implementation phase)
- **OQ-3**: Widget layout grid spec — number of columns, responsive breakpoints (deferred to UX phase)
