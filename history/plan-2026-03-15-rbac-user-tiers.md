# Plan: DEFT Evolution — User Tiers & Role-Based Access Control

**Date**: 2026-03-15
**Branch**: spec/rbac-user-tiers
**Status**: Approved — ready for implementation via `rbac-user-tiers.yml` workflow
**Spec**: SPECIFICATION-rbac-user-tiers.md
**PRD**: PRD-rbac-user-tiers.md
**vBRIEF**: vbrief/specification-rbac-user-tiers.vbrief.json

---

## Interview Decisions

| # | Question | Decision |
|---|---|---|
| 1 | What is the public doc/wiki system? | All content types are tagged threads. Wiki/docs/tutorials = threads with `thread_type` tag. No separate docs site. |
| 2 | What is the chatbot? | AI-first with live agent handoff via IO channels (option 3). |
| 3 | How is DEFT's internal team structured? | One `deft` org with department spaces: deft-sales, deft-support, deft-finance. |
| 4 | How do DEFT employees see cross-org tickets/leads? | Four global spaces (global-docs, global-forum, global-support, global-leads) with filtered views per audience. |
| 5 | What does provisioning mean? | Placeholder only in this phase. Billing provider deferred. |
| 6 | How does Tier 2 → Tier 3 conversion work? | All three paths: self-service, sales-assisted, admin override. |
| 7 | Global space taxonomy? | Four named spaces: global-docs (public), global-forum (public), global-support (auth, org-filtered), global-leads (DEFT-only). |
| 8 | Tier resolution when user qualifies for multiple tiers? | Highest-privilege-wins. User can customize home screen widget layout. |
| 9 | Tier 4 DEFT employee home screen? | Department-specific defaults based on deft org space membership. |
| 10 | Anonymous lead identity resolution on registration? | Clerk anonymous session promoted to full account; prior sessions linked automatically. |

---

## Implementation Phases

### Phase 1: Data Model & Backend Foundation
- Schema: thread_type, visibility, org_id on threads; user_home_preferences table; anon_session_id on leads
- Seed: 4 global spaces + deft org + 3 dept spaces
- Backend: ResolveTier service, /api/me/tier, /api/me/home-preferences
- Dependencies: none

### Phase 2: Frontend Infrastructure
- useTier() hook, Next.js middleware route protection, public route group
- Widget + HomeLayout + HomeLayoutEditor + useHomeLayout
- Dependencies: Phase 1

### Phase 3: Tier 1 & Tier 2 Home Screens *(parallel: T1 and T2)*
- Tier 1: DocsHighlights, ForumHighlights, GetStarted, chatbot
- Tier 2: Profile, ForumActivity, SupportTickets, UpgradeCTA
- Actions: forum post create, support ticket create for Tier 2+
- Dependencies: Phase 2

### Phase 4: Tier 3 & Tier 5 Home Screens *(parallel with Phase 5)*
- Tier 3 member: OrgOverview, OrgSupportTickets, forum activity
- Tier 3 owner: OrgSupportDashboard, BillingStatus stub
- Tier 5: OrgAccessControl, OrgRBACEditor
- Dependencies: Phase 3

### Phase 5: Tier 4 & Tier 6 Home Screens *(parallel with Phase 4)*
- Tier 4 Sales: LeadPipeline, RecentLeads, ConversionMetrics
- Tier 4 Support: TicketQueue, TicketStats
- Tier 4 Finance: BillingOverview
- Tier 6: SystemHealth, RecentAuditLog, all T4 widgets
- Dependencies: Phase 2

### Phase 6: Chatbot *(parallel with Phases 4+5)*
- ChatbotWidget (async, Clerk anon token)
- POST /api/chat/message (stubbed AI → RAG)
- Lead capture, live agent handoff, anon session promotion
- Dependencies: Phase 3 (anon sessions), Phase 1 (global-docs)

### Phase 7: Conversion Flows & Customization *(2 agents in parallel)*
- Conversion: self-service stub, sales-assisted, admin override
- Customization: drag-drop polish, reset-to-defaults, E2E persistence
- Dependencies: Phases 3–6 (all must complete)

---

## Dependency Graph

```
1 → 2 → 3 → 4
            6
      2 → 5
{4 + 5 + 6} → 7
```

Auto-chained via `.github/workflows/rbac-user-tiers.yml`.
Dispatch Phase 1 to start: `gh workflow run rbac-user-tiers.yml -f phase=1`
