# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Forum, Docs, Support, and Notifications navigation pages (web)
- Tier-aware home screen (Tier 4 and Tier 6 variants)
- Admin UI completeness: org detail, user detail, impersonation, platform settings, security audit, RBAC override, API usage, LLM usage, exports
- Reporting module: sales pipeline funnel, deal velocity, support metrics, org breakdown
- RBAC/user tier system with FlexPoint billing
- IO channels: email (IMAP/SMTP), voice (LiveKit), chat widget
- GDPR data export and erasure workflows
- Webhook subscriptions and delivery
- Full-text search across threads, messages, orgs
- Audit log for all mutating operations
- Multi-provider LLM integration
- Real-time WebSocket hub with per-org broadcast
- Admin impersonation with audit trail
- Feature flags per org
- Fuzz test suites across 14 packages
- vBRIEF specification files for all major features

### Fixed
- Support ticket cards, status badges, stats strip, and error banners now use dark-shifted color palettes in night/dark mode for readable text contrast

### Changed
- Updated deft submodule URL to `deftai/directive` (previously `visionik/deft`) and pinned to latest master
- Updated vBRIEF repo reference to `deftai/vBRIEF` (previously `visionik/vBRIEF`); spec version remains 0.5

[Unreleased]: https://github.com/abraderAI/crm-project/compare/main...HEAD
