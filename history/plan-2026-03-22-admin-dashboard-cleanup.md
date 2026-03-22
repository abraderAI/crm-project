# Admin Dashboard Cleanup — Hybrid A+C

**Date**: 2026-03-22
**Branch**: fix/admin-home-widgets

## Summary

Removed fake/stub data from both the `/admin` home page and the tier-6 user home screen.
Replaced with real data from existing reporting endpoints and admin quick links.

## Changes

### /admin home page (page.tsx)
- Removed: entity stats grid (Orgs, Users, Threads, Messages) — raw DB counts that conflated support tickets
- Removed: system health cards (DB Size, Failed Webhooks, Pending Notifications)
- Removed: PlatformStats KPI row (Total Orgs, Users, Threads, DB Size, API Uptime)
- Added: support snapshot (Open Tickets, Overdue, Avg Resolution) from `GET /admin/reports/support`
- Added: sales snapshot (Total Leads, Win Rate, Avg Deal Value) from `GET /admin/reports/sales`
- Added: quick links to Organizations, Users, Audit Log, Feature Flags, Settings

### Tier-6 user home (tier-6-home.tsx, tier6-home-screen.tsx)
- Removed: all 8 stub widgets (SystemHealth, RecentAuditLog, LeadPipeline, RecentLeads, ConversionMetrics, TicketQueue, TicketStats, BillingOverview)
- Added: admin quick links pointing to /admin, reports, users, feature flags, audit log
- Updated: TIER_6_DEFAULT layout to empty array

### widget-api.ts
- Removed: 8 stub fetch functions that returned hardcoded/fabricated data
- Kept: type interfaces for future use when real endpoints are ready

### No backend changes required
All data comes from existing endpoints.
