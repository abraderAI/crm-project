# Sales CRM Leads Interface — PRD

**Version:** 1.0
**Date:** 2026-03-23
**Source Spec:** v1.4 (March 2026)
**Branch:** feat/sales-crm

---

## Problem Statement

DEFT's sales team currently manages pre-sales leads as generic Threads within the platform. This approach lacks:
- Distinct Company, Contact, and Opportunity entity types with separate visibility rules
- Record-level access control (platform RBAC is currently board/space-scoped only)
- Integrated email communication (customer email handled entirely outside the platform)
- Structured follow-up task management with reminders and overdue flagging
- Sales reporting, pipeline forecasting, and executive pipeline visibility

DEFT sales organization members need a purpose-built CRM interface within the existing platform to manage the full B2B sales lifecycle — from initial lead capture through opportunity tracking to closed-won conversion.

---

## Goals

**Primary:** Deliver a complete internal Sales CRM for DEFT sales team members within the existing platform, covering Company/Contact/Opportunity management, integrated email, task tracking, and sales reporting.

**Secondary:**
- Give the DEFT CEO real-time, full-visibility pipeline and forecast reporting via a dedicated executive dashboard
- Replace external email tools for all customer-facing sales communication
- Provide pipeline forecasting and performance analytics for sales managers and administrators
- Enable bulk lead creation via CSV import

**Non-Goals (explicitly out of scope for this phase):**
- Custom field admin UI (fields stored in metadata JSON; admin configuration UI deferred)
- PDF report export (CSV export only in this phase)
- Multi-currency support (USD only)
- Quota target management and pipeline coverage ratio calculation
- Mobile app or native mobile support
- Public-facing lead capture forms or inbound marketing automation
- External CRM integrations (Salesforce, HubSpot, etc.)

---

## User Stories

**Sales Representative:**
- As a sales rep, I want to create and manage Company and Contact records so I can organize my prospects in one place.
- As a sales rep, I want to track Opportunities through a configurable sales pipeline so I can manage my deals to close.
- As a sales rep, I want to send and receive emails from within the CRM so I never need an external mail client for customer communication.
- As a sales rep, I want to create follow-up tasks with due dates and receive reminders so I never miss a commitment.
- As a sales rep, I want my Contacts and Opportunities to be private by default so my pipeline is not visible to other reps.
- As a sales rep, I want to import Companies and Contacts via CSV so I can bulk-load prospect lists efficiently.
- As a sales rep, I want to see pipeline and activity reports filtered to my own records.

**Sales Administrator:**
- As a sales admin, I want full visibility and edit access to all Company, Contact, Opportunity, and Task records across the team.
- As a sales admin, I want to assign or reassign any record to any team member.
- As a sales admin, I want to configure pipeline stages and default probabilities for the org.
- As a sales admin, I want to see full activity and performance reporting across all users.

**DEFT CEO:**
- As the CEO, I want a dedicated "Show Me the Money" executive report showing all deals closing soon with real-time pipeline data so I have clear revenue visibility.
- As the CEO, I want full visibility across all pipeline across all owners without member-level filtering.

---

## Requirements

### Functional Requirements

#### Entity Model

**FR-1:** The system SHALL model Company, Contact, and Opportunity as distinct CRM entity types. Each SHALL be stored as a Thread with a `crm_type` metadata field (`company`, `contact`, `opportunity`) within a single CRM-type Space per org containing three fixed boards: `companies`, `contacts`, and `opportunities`.

**FR-2:** The system SHALL maintain a `crm_links` join table storing typed directional relationships between CRM entity Threads. Link types SHALL be: `contact_company` (Contact → Company), `opportunity_company` (Opportunity → Company), and `opportunity_contact` (Opportunity → Contact, with an `is_primary` boolean flag). A single Opportunity MAY have one primary Contact and multiple secondary Contacts.

#### Company Records

**FR-3:** Company records SHALL store the following core attributes in Thread metadata: name (required), industry, company size, location, website, status, and free-text description.

**FR-4:** The Company view SHALL display: all core attributes, linked Contacts (subject to visibility rules), all historical activities/emails/tasks/notes associated with the Company or its Contacts (subject to visibility rules), an aggregated Opportunity summary (count and high-level status of visible Opportunities only), and one-click navigation to any linked Opportunity the requesting user has permission to access.

**FR-5:** Company records SHALL be visible to any DEFT sales org member who has at least one active Opportunity, owned Contact, or assigned Task linked to that Company. Edit permissions on Company-level attributes SHALL be restricted to the Company owner and administrators.

#### Contact Records

**FR-6:** Contact records SHALL store the following core attributes: name (required), title, phone number, email address, LinkedIn URL, notes, and a link to the parent Company (via `crm_links`).

**FR-7:** The Contact view SHALL display: all core attributes, parent Company details with one-click navigation, all historical activities/emails/tasks/notes specific to that Contact, and linked Opportunities where this Contact appears as primary or secondary (subject to visibility rules).

**FR-8:** Contact records SHALL be private by default. A Contact SHALL be visible only to: the Contact owner, the owner(s) of any Opportunity linked to that Contact, the assignee(s) of any Task linked to that Contact, and DEFT sales org administrators.

#### Opportunity Records

**FR-9:** Opportunity records SHALL store the following core attributes: name/title (required), associated Company (required, via `crm_links`), associated Contact(s) (optional; via `crm_links` with `is_primary` flag), pipeline stage, effective probability (see FR-10), deal amount in USD, expected close date, weighted forecast value (auto-calculated: amount × effective probability / 100), opportunity type (New Business / Expansion / Renewal), lead source, and opportunity owner.

**FR-10:** Pipeline stages SHALL use the existing configurable stage system. Each stage SHALL carry a default probability percentage: New Lead 5%, Contacted 15%, Qualified 30%, Proposal 50%, Negotiation 75%, Closed Won 100%, Closed Lost 0%, Nurturing 10%. Sales reps MAY set a per-Opportunity probability override stored in Thread metadata; if set, it SHALL be used instead of the stage default for weighted forecast calculations. Administrators MAY configure stage default probabilities per org via the existing pipeline config in org metadata.

**FR-11:** The Opportunity view SHALL display: all core attributes, ownership and assignment, a chronological activity timeline (emails sent/received, tasks, notes, calls, attachments), a visual pipeline stage indicator with days-in-stage and opportunity age (days since creation), an overdue indicator when expected close date has passed, related open tasks (Opportunity-specific and inherited from linked Company/Contact), and one-click navigation to the parent Company and primary Contact.

**FR-12:** Opportunity records SHALL be private by default. An Opportunity SHALL be visible only to: the Opportunity owner, the assignee(s) of any Task linked to that Opportunity, and DEFT sales org administrators.

**FR-13:** The following actions SHALL be available directly from the Opportunity view:
- Advance to next stage (optional comment)
- Move to any stage (reason required for backward movement)
- Mark as Closed Won or Closed Lost (close/loss reason required)
- Create a new follow-up task
- Compose and send email via the embedded editor
- Add a note
- Attach a file
- Reassign ownership
- Edit Opportunity details

#### Record-Level ACL

**FR-14:** The system SHALL implement a `thread_acl` table that grants per-thread read visibility to specific users. ACL grants SHALL be automatically maintained by: record ownership (owner of Company/Contact/Opportunity), task assignment (assignee of any `crm_tasks` row with matching parent thread), and linked entity ownership (Contact ACL includes owners of all Opportunities linked to it via `crm_links`). All mutations to records, tasks, and `crm_links` SHALL trigger synchronous ACL recalculation within the same DB transaction.

**FR-15:** ACL SHALL be enforced at the repository query layer. All list and get operations for Company, Contact, and Opportunity Threads SHALL join against `thread_acl` and filter to rows where the requesting user has a grant, unless the user is a DEFT sales org administrator or the configured CEO user.

#### Manual Lead Entry

**FR-16:** The system SHALL provide guided creation forms for Company and Contact records. Company and Contact creation SHALL have separate but linked entry flows. During Contact creation, the user SHALL be able to associate the new Contact to an existing Company record.

**FR-17:** All manual entry forms SHALL perform real-time duplicate detection: name matching for Companies and email address matching for Contacts. A duplicate warning SHALL be displayed before save, requiring explicit user confirmation to proceed.

#### CSV Import

**FR-18:** The system SHALL provide a CSV bulk import function for Companies and Contacts. The import SHALL be a two-step process:
1. Upload + validate: server parses the CSV, returns a preview payload containing valid rows, error rows (with per-row error detail), and duplicate detections.
2. Confirm + apply: user reviews the preview and confirms; the server applies all valid rows within a single DB transaction. On any failure, the entire batch is rolled back.

**FR-19:** CSV import validation SHALL check: required field presence, email address format, duplicate detection against existing records, and field length limits. Import history SHALL be logged with: importing user ID, timestamp, entity type, total row count, success count, error count, and per-row error details.

#### Follow-up Tasks

**FR-20:** The system SHALL provide a `crm_tasks` table for follow-up tasks attachable to Company, Contact, or Opportunity entities. Task attributes SHALL include: parent entity type and thread ID, title (required), description, assigned user ID, due date, priority (low / medium / high / urgent), and status (open / in-progress / completed / cancelled).

**FR-21:** Tasks SHALL be visible only to: the assigned user, the owner of the parent Company/Contact/Opportunity record, and DEFT sales org administrators. Task history SHALL be immutable; completed or cancelled tasks SHALL not be editable.

**FR-22:** The system SHALL deliver task notifications through the existing `NotificationProvider` abstraction (in-app via DB + WebSocket and email via Resend) for the following events: task assigned to a user (immediate), task due the following day (24 hours before due date), and task overdue (on the due date, then daily until resolved or cancelled). Users MAY configure task notification channel preferences via the existing notification preferences page.

**FR-23:** All task mutations (create, update, assign, status change) SHALL be captured by the existing audit log system.

#### Ownership & Assignment

**FR-24:** Any DEFT sales org member SHALL be able to assign or reassign ownership of any Company, Contact, or Opportunity record to another DEFT sales org member, subject to admin approval if configured. Opportunity ownership SHALL be assignable independently of its parent Company or Contact ownership.

**FR-25:** All ownership transfers SHALL be recorded in the audit log with before/after user IDs and timestamps. ACL grants SHALL be recalculated synchronously on ownership transfer.

**FR-26:** Default ownership on record creation SHALL be the creating user. On CSV import, ownership SHALL default to the importing user unless a valid `owner_email` field matching an existing DEFT sales org member is provided in the CSV row.

#### Email Integration

**FR-27 — Per-Rep Gmail Configuration via Channel Settings:**
Email integration SHALL be built on the existing channel infrastructure (`EmailInbox` model and `ChannelGateway`). A new `sales_crm` routing action SHALL be added to the `RoutingAction` enum. A DEFT sales org administrator SHALL first register each rep's `@deft.co` address as an `EmailInbox` record with `routing_action = "sales_crm"` and `assigned_user_id` set to the rep's Clerk user ID in system channel settings. The rep SHALL then connect their personal Gmail credentials via the CRM profile settings OAuth2 flow, which stores an encrypted OAuth2 refresh token in the pre-existing `EmailInbox` record. A rep whose inbox has not been pre-registered by an admin SHALL receive a clear prompt to contact their admin. Outbound emails SHALL be sent from the rep's personal `@deft.co` address. Inbound monitoring SHALL be established via Gmail Pub/Sub per connected inbox. A rep without a connected OAuth token SHALL not be able to send or receive email through the CRM.

**FR-28:** Every outbound email composed and sent through the CRM SHALL be persisted as a Message with `type = "email"` on the relevant Company, Contact, or Opportunity Thread and linked to the sending rep's user ID. If an email is sent from within a specific entity view, it SHALL be linked to that entity via the activity timeline.

**FR-29 — Per-Rep Inbound Email Routing:**
Every inbound email received on a connected rep's `@deft.co` account and delivered via that rep's Gmail Pub/Sub subscription SHALL be automatically matched to existing Company, Contact, and/or Opportunity records by sender email address and stored as a Message with `type = "email"` on matching Thread(s). Inbound emails that cannot be automatically matched SHALL be placed in that rep's unassigned inbox queue for manual routing.

**FR-30:** The system SHALL provide a rich-text email composer (using the existing Tiptap editor) embedded within Company, Contact, and Opportunity views. The composer SHALL support: bold, italic, bullet/numbered lists, tables, inline images, hyperlinks, file attachments (from existing CRM uploads or local device), email signature management, and an email template library.

**FR-31:** DEFT sales org members SHALL NOT require or be permitted to use external email clients for customer-facing sales communication once their `@deft.co` account is connected. All customer communication SHALL be routed through and permanently stored within the CRM.

**FR-32a — `sales@deft.co` Lead Generation Inbox:**
An administrator SHALL configure `sales@deft.co` as an `EmailInbox` record in system channel settings with `routing_action = "sales_lead"` (the existing routing action already supported by the `ChannelGateway`). Every inbound email received at `sales@deft.co` SHALL automatically upsert a Lead record with: sender email address, sender name (from email headers), source = `"email"`, status = `"anonymous"`, and the email body stored in the Lead's metadata. Deduplication SHALL be performed by sender email address — subsequent emails from the same sender update the existing Lead rather than creating a new one. A notification SHALL be sent to all DEFT sales org administrators alerting them of a new or updated inbound lead. Admins MAY assign ownership of the Lead from the notification or lead list.

#### Reports

**FR-32:** The system SHALL include a dedicated Reports page accessible to all DEFT sales org members. DEFT sales org administrators and the user identified by `DEFT_CEO_USER_ID` SHALL see all reports unfiltered across all records and users. All other members SHALL see reports filtered to their own owned/assigned records (Companies they own, Contacts they own, Opportunities they own, Tasks assigned to them).

**FR-33 — Pipeline Reports:**
- Pipeline by Stage: count and USD value of open Opportunities per stage, plus weighted forecast value, number of deals, and average deal size per stage.
- Pipeline by Owner: total pipeline value, weighted forecast value, and deal count per DEFT sales org member.
- Pipeline by Expected Close Date: Opportunities grouped by month/quarter of expected close date, showing open value and weighted forecast.
- Overdue Opportunities: list of Opportunities past expected close date that remain open.

**FR-34 — Forecasting Reports:**
- Forecast by Owner: weighted forecast value per owner for current month, current quarter, and current year.
- Forecast by Stage: weighted forecast contribution per stage.
- Forecast Accuracy: historical comparison of prior weighted forecast vs. actual closed-won amount for completed periods.

**FR-35 — Activity & Productivity Reports:**
- Activity Summary by User: count of emails sent, tasks created, tasks completed, notes added, and calls/meetings logged per user over a selected period.
- Tasks Overdue: list of overdue tasks grouped by assignee, filterable by priority and due date range.
- Tasks Completed: count and completion rate of tasks by assignee and time period.

**FR-36 — Performance & Conversion Reports:**
- Win Rate: percentage of Opportunities marked Closed Won vs. total closed, segmented by owner, lead source, opportunity type, and time period.
- Average Sales Cycle: average days from Opportunity creation to close (won or lost), segmented by owner, stage progression, or deal size.
- Conversion by Stage: percentage of Opportunities advancing from each stage to the next.
- Source Performance: Opportunity count and closed-won value by lead source.

**FR-37 — Company & Contact Reports:**
- Companies by Status: count and list of Companies grouped by status field.
- Contacts by Company: summary of Contact count per Company.
- Recently Created/Modified Records: lists of new or recently updated Companies, Contacts, and Opportunities.

**FR-38 — Cross-Cutting Report Features:**
All reports SHALL support:
- Date range filters: custom range, this week/month/quarter/year, last X days
- Filtering by owner, stage, opportunity type, lead source, and other key attributes where applicable
- Sortable columns and search within results
- CSV export of visible data
- Prominent summary metrics (totals, counts, averages, percentages)
- Visual elements (charts, graphs, progress bars) where they meaningfully enhance understanding
- Real-time data refresh on page load or manual refresh trigger

**FR-39 — "Show Me the Money" Executive Report:**
The system SHALL provide a dedicated executive report accessible exclusively to users identified by `DEFT_CEO_USER_ID` (and optionally other designated administrators). The report SHALL present two sections:

*Section 1 — Closing Deals & Forecast Summary:*
- Summary metric cards: total weighted forecast closing this month, total weighted forecast closing this quarter, total unweighted pipeline closing this quarter, number of deals closing within 30/60/90 days, count of overdue/at-risk deals
- Deals closing soon table (columns: Opportunity name, Company, primary Contact, stage, stage probability, weighted value, deal amount, expected close date, owner name, days until close / overdue days); sorted by default: expected close date ascending, then weighted value descending
- Pipeline funnel or bar chart showing weighted/unweighted value by close date period
- Total closing amount displayed prominently at the page bottom

*Section 2 — Broader Pipeline Overview:*
- All open Opportunities table with same columns as above plus opportunity age (days since creation) and days in current stage
- Summary stats: total open count, total pipeline value (unweighted), total weighted pipeline value, breakdown by stage (count + value), breakdown by owner (count + value + weighted value), top 10 Opportunities by deal amount, stalled Opportunities (no activity in last 30 days, configurable per org via metadata)

**FR-40:** The "Show Me the Money" report SHALL show data across all owners with no member-level visibility filtering. It SHALL support: real-time data refresh on load, date range selector (default: current month + next month; options for current quarter and custom range), quick filters by owner/stage/close date range/company, and CSV export of all data. Owner names SHALL be prominently and clearly displayed.

#### AI & LLM Features

**FR-45 — Personal AI Sales Briefing:**
The system SHALL provide a personal AI briefing for each DEFT sales org member. The briefing SHALL be available via an on-demand "Brief Me" button in the CRM dashboard, and optionally via a scheduled daily push notification (opt-in via notification preferences). When triggered, the LLM SHALL scan the requesting user's open leads, Opportunities, Tasks, and recent activity and return a prioritized briefing covering: what is happening across their pipeline, what needs immediate attention, and recommended next actions. Output SHALL be displayed in a dedicated panel or modal. The existing `LLMProvider` interface SHALL be extended to support this briefing context.

**FR-46 — Inbound Email AI Summary:**
The system SHALL automatically analyze every inbound email delivered via the Gmail Pub/Sub webhook using the LLM. The LLM SHALL be provided the email content plus the context of the associated CRM record (Company, Contact, and/or Opportunity Thread including recent activity). The LLM SHALL produce a 1–2 line summary of the email highlighting urgency, sentiment, and recommended action. This summary SHALL be delivered immediately to the assigned sales rep as an in-app system notification (and email if the rep's preferences include email for notifications). The summary SHALL help reps prioritize hot inbound messages without reading every email.

**FR-47 — CEO Strategic Pipeline Analysis:**
The "Show Me the Money" executive report SHALL include an on-demand LLM-generated strategic analysis section. When the CEO triggers it, the LLM SHALL analyze all open Opportunities (particularly those closing soon), identify patterns, risks, and leverage points, and produce a strategic narrative outlining key moves the CEO could take to help close deals. Output SHALL be displayed as a dedicated section within the report. The CEO MAY regenerate the analysis at any time.

**FR-48 — "Close This Deal Now" Strategy:**
The system SHALL provide a "Close This Deal Now" action available from any Opportunity view. When triggered, the LLM SHALL perform a focused scan of the Opportunity's full record context: Thread metadata, all Messages (emails, notes, call logs), linked Company and Contact records, open Tasks, stage history, and days-in-stage. The LLM SHALL return a targeted closing strategy specific to that deal. Output SHALL be displayed in a slide-over panel within the Opportunity view. The sales rep MAY optionally save the strategy as a pinned note on the Opportunity Thread. This action SHALL also be invocable from the AI Insights briefing panel (FR-45) for any listed Opportunity.

**FR-49 — CRM Data Quality Enforcement:**
The system SHALL enforce CRM record completeness using a two-layer approach:
- *Detection layer (rule-based):* Configurable completeness rules SHALL run on record save and on a nightly scheduled scan. Default rules SHALL flag: Opportunity missing deal amount or expected close date after 48 hours of creation; Opportunity missing a linked Company; Opportunity with no activity logged in the last 14 days (configurable per org); Contact missing email address or phone number.
- *Notification layer (LLM-written):* When a violation is detected, the LLM SHALL generate a contextual, pointed notification message specific to the record's gaps. The message SHALL be delivered to the record owner as an in-app notification. A daily digest of all violations across the org SHALL be sent to DEFT sales org administrators. The LLM message tone SHALL be direct and constructive ("scolding" as appropriate) while remaining professional.

#### Global Features

**FR-41:** The system SHALL provide global search across Company, Contact, Opportunity, Task, and Email history records, respecting ACL visibility rules.

**FR-42:** The system SHALL provide the following list views with filtering, sorting, and grouping: All Companies, All Contacts, Pipeline (all accessible Opportunities), My Opportunities, and Team Opportunities (admins/CEO only).

**FR-43:** Closed Opportunities (Closed Won or Closed Lost) SHALL remain visible and searchable but SHALL be read-only for all fields except: adding notes, adding attachments, and creating post-close tasks. This behavior SHALL be configurable per org.

**FR-44:** All create, update, assign, stage change, close, and delete actions on any CRM entity SHALL be captured by the existing audit log system with user ID, timestamp, entity type/ID, and before/after state JSON.

**FR-44a — Closed Won Support & Finance Handoff:**
When an Opportunity is marked as Closed Won, the system SHALL automatically trigger a handoff workflow:
- A support ticket SHALL be created in the existing support ticket system (`api/internal/support`) containing: Opportunity name and ID, deal amount, close date, close reason, linked Company record details (name, industry, size, location, website, status), primary Contact details (name, title, email, phone), a summary of recent Opportunity activity (last 5 messages), and a direct link back to the Opportunity Thread.
- The ticket SHALL be assigned to the support team and a copy notification SHALL be sent to the finance team (both identified by configurable org-level metadata fields `support_team_user_ids` and `finance_team_user_ids`).
- The Opportunity owner SHALL receive a confirmation notification that the handoff ticket was created, including a link to the ticket.
- Closed Lost opportunities SHALL NOT trigger a handoff.
- If ticket creation fails, the Closed Won transition SHALL still succeed and the failure SHALL be logged to the audit log and surfaced as an admin alert.

---

### Non-Functional Requirements

**NFR-1 — Security:** Record-level ACL SHALL be enforced at the repository query layer on every read operation. No CRM entity SHALL be returned to a user without an explicit ACL grant or admin/CEO role. Gmail OAuth2 credentials SHALL be stored in `secrets/` per project conventions. No secrets SHALL appear in code or version control.

**NFR-2 — Performance:** All entity list views SHALL respond within 2 seconds for datasets up to 10,000 records with ACL filtering applied. The `thread_acl` table SHALL be indexed on `(user_id, thread_id)`. Reports SHALL complete within 5 seconds for up to 12 months of data.

**NFR-3 — Test Coverage:** All new backend packages SHALL achieve ≥85% test coverage. ACL enforcement paths SHALL have explicit unit tests covering every grant/deny scenario. CSV import validation and Gmail integration SHALL have integration tests using mocked external services. Fuzz testing (≥50 inputs per target) SHALL be applied to CSV parsing and ACL query construction.

**NFR-4 — Audit Completeness:** Every mutation on Company, Contact, Opportunity, Task, `crm_links`, and `thread_acl` SHALL produce an immutable audit log entry capturing actor, action, entity type/ID, and before/after JSON state.

**NFR-5 — Concurrency:** ACL recalculation triggered by ownership changes, task assignments, and `crm_links` mutations SHALL execute atomically within the same DB transaction as the triggering mutation. No visibility gap SHALL be possible between the mutation and ACL update.

**NFR-6 — Compatibility:** This implementation SHALL extend the existing Thread/Message/RBAC infrastructure. No breaking changes to existing API contracts are permitted. All new API endpoints SHALL be versioned under `/v1/`.

**NFR-7 — Responsiveness:** All frontend views SHALL support desktop and tablet viewports. Mobile-specific layout is not required in this phase.

**NFR-8 — LLM Cost & Latency:** LLM calls SHALL be reserved for value-add actions (briefings, email summaries, strategies, notifications) and SHALL NOT be triggered on every page load or record view. Inbound email LLM analysis (FR-46) and data quality notifications (FR-49) SHALL be processed asynchronously and SHALL NOT block the user-facing request. On-demand LLM features (FR-45, FR-47, FR-48) SHALL display a loading state and complete within 15 seconds. The existing `LLMProvider` interface (GrokProvider default) SHALL be used for all LLM calls; no direct API calls to LLM providers are permitted outside this abstraction.

---

## Success Metrics

- DEFT sales team members can create, view, and manage Company, Contact, and Opportunity records with correct private/shared visibility enforced at the API layer
- All customer email sent and received exclusively through the CRM; zero reliance on external email clients
- CEO can access "Show Me the Money" report with real-time, unfiltered pipeline data
- All pipeline stage transitions, weighted forecast calculations, and report aggregations produce mathematically correct results
- ≥85% test coverage on all new backend packages with all ACL paths explicitly tested
- No existing platform features broken (full existing test suite passes)
- AI Sales Briefing, "Close This Deal Now", and CEO strategic analysis return coherent, record-specific LLM output within 15 seconds
- Inbound email LLM summaries are delivered as notifications asynchronously without blocking email ingestion
- CRM data quality violations are detected on save and nightly; LLM-written notifications are delivered to reps and admin digest

---

## Open Questions

- **Gmail OAuth2 mechanism:** Whether `sales@deft.co` uses a Google Workspace service account with domain-wide delegation or a shared OAuth2 refresh token. To be resolved with infrastructure team before Phase 3 implementation begins.
- **Stalled opportunity threshold:** Default 30 days with no activity; configurable per org via org metadata. To be confirmed in specification.
- **Email template library:** Number and categories of initial templates to ship. To be defined during frontend implementation phase.
- **Quota targets:** Pipeline coverage ratio (total weighted pipeline vs. quota) requires quota configuration per owner/period. Quota management is deferred; the coverage ratio metric is omitted from initial reports.
- **Gmail OAuth2 Google Workspace setup:** Per-rep OAuth requires each `@deft.co` account to be part of a Google Workspace organization with Gmail API and Pub/Sub enabled. Confirm Google Workspace org configuration and required API scopes with infrastructure team before Phase 3.
- **Support & finance team identification:** FR-44a uses `support_team_user_ids` and `finance_team_user_ids` in org metadata. Confirm whether these are individual user IDs or team/role identifiers, and who configures them (admin UI or direct config).
- **`sales@deft.co` OAuth ownership:** The shared lead generation inbox requires its own OAuth2 connection. Confirm which Google Workspace admin account owns and maintains this connection.

---

*Generated from interview decisions on 2026-03-23. Source spec: v1.4 (March 2026).*
*Next step: user approval → `vbrief/specification-sales-crm.vbrief.json` → `SPECIFICATION-sales-crm.md`*
