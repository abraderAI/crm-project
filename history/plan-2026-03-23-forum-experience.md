# Forums Feature: DEFT General Discussion + Admin Management

## Problem Statement

The `/forum` page is non-functional:
- "Post a Thread" links to `/sign-in` which redirects authenticated users to home
- No thread detail view (stub only)
- No thread creation UI for authenticated users
- No forum seed data
- No admin management for forums
- The forum UI is minimal and uninviting

## Proposed Changes

1. **Branch**: `feat/forum-experience` from `main`
2. **Seed Data**: 6-8 threads about deftai/directive in global-forum space
3. **Forum Index**: Modern card-based layout with header, sort, pinned threads
4. **Thread Detail**: Full thread view with body, votes, author info
5. **Thread Creation**: Authenticated `/forum/new` route with rich editor
6. **Admin Management**: `/admin/forums` page for moderation
7. **Visual Design**: Rounded cards, gradients, avatars, dark mode
8. **Tests**: ≥85% coverage for all new code
