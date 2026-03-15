Implement IO Phase 7 (Integration & Deploy) of the DEFT Evolution CRM project (abraderAI/crm-project).

START BY reading AGENTS.md in the repo root — it contains all required framework reading and
coding standards you MUST follow before writing any code.

---

## CONTEXT

The IO Channels add-on module is complete on branch `feat/io-phase1-channel-gateway`.
All phases 1–6 have been merged into that branch:

- Phase 1: Channel Gateway & Infrastructure
- Phase 2: Inbound Email
- Phase 3: Voice / LiveKit
- Phase 4: AI Web Chat Widget (embeddable IIFE — `widget/`)
- Phase 5: Agentic CLI (`api/cmd/cli/`, `api/internal/cli/`, binary `api/bin/deft`)
- Phase 6: Channel Admin UI (`web/src/app/admin/channels/`)

**YOUR WORKING BRANCH**: check out `feat/io-phase1-channel-gateway`, then create
`feat/io-phase7-integration-deploy` from it.

---

## YOUR TASK

### 1. Environment Variables

**`api/.env.example`** — add these new IO env vars with placeholder values:
```
# IO Channel Gateway
CHAT_JWT_SECRET=change-me-in-production-min-32-chars
INTERNAL_API_KEY=change-me-in-production

# LiveKit (Voice channel)
LIVEKIT_URL=wss://your-livekit-server.example.com
LIVEKIT_API_KEY=your-livekit-api-key
LIVEKIT_API_SECRET=your-livekit-api-secret
LIVEKIT_WEBHOOK_TOKEN=your-livekit-webhook-token

# Agentic CLI (server-side — optional)
# CLI credentials are stored client-side in ~/.deft-cli.yaml
```

**`docs/env-vars.md`** — add a new "IO Channels" section documenting each var:
- Name, description, required/optional, default (if any), example value
- Note which vars must be overridden in production (CHAT_JWT_SECRET, INTERNAL_API_KEY)
- Note that LiveKit vars are only required when the voice channel is enabled

### 2. Docker Compose (`docker/docker-compose.yml`)

Add the new IO env vars to the `api` service environment block, reading from host env
(same pattern as existing CLERK_* vars):
```yaml
- CHAT_JWT_SECRET=${CHAT_JWT_SECRET:-change-me-in-production}
- INTERNAL_API_KEY=${INTERNAL_API_KEY:-change-me-in-production}
- LIVEKIT_URL=${LIVEKIT_URL:-}
- LIVEKIT_API_KEY=${LIVEKIT_API_KEY:-}
- LIVEKIT_API_SECRET=${LIVEKIT_API_SECRET:-}
- LIVEKIT_WEBHOOK_TOKEN=${LIVEKIT_WEBHOOK_TOKEN:-}
```

Also add a `widget` service that serves `widget/dist/widget.js` as a static file on
port 4000 (use `nginx:alpine` with a single `location /` that serves the dist dir).
This is optional for local dev but required for E2E testing.

### 3. GitHub Actions (`.github/workflows/implement.yml` or create new file)

Create `.github/workflows/io-channels.yml` — a dedicated CI workflow for the IO add-on:

```yaml
name: IO Channels CI
on:
  push:
    branches: [feat/io-phase1-channel-gateway, feat/io-phase*]
  pull_request:
    branches: [feat/io-phase1-channel-gateway, main]

jobs:
  api:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - name: Install task
        run: sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin
      - name: Install golangci-lint
        run: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /usr/local/bin
      - run: task fmt
      - run: task lint
      - run: task test:coverage
      - run: task cli:build

  widget:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: '20' }
      - name: Install task
        run: sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin
      - run: cd widget && npm ci
      - run: task widget:build
      - run: task widget:test:coverage
      - uses: actions/upload-artifact@v4
        with:
          name: widget-dist
          path: widget/dist/

  web:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: '20' }
      - name: Install task
        run: sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin
      - run: cd web && npm ci
      - run: task web:fmt:check
      - run: task web:lint
      - run: task web:typecheck
      - run: task web:test:coverage
```

### 4. Playwright E2E Integration Tests (`web/e2e/`)

Add `io-channels.spec.ts` with the following test cases (use mocked API responses
via `page.route()` — do not require a live backend):

**Test A — Channel overview page**
  - Navigate to /admin/channels
  - Mock GET /v1/orgs/*/channels/email|voice|chat health endpoints
  - Assert all 3 channel cards are visible
  - Assert health badges render with correct colours

**Test B — Channel config form**
  - Navigate to /admin/channels/email
  - Mock GET config + health endpoints
  - Assert config form fields are present (imap_host, imap_user, etc.)
  - Assert secret fields render as "••••••••"
  - Fill in imap_host, click Save
  - Mock PUT endpoint returning 200
  - Assert success toast/notification appears

**Test C — DLQ monitor**
  - Navigate to /admin/channels/email/dlq (or use DLQ section of config page)
  - Mock GET DLQ endpoint returning 2 failed events
  - Assert event rows appear in table
  - Click "Retry" on first row
  - Mock POST retry endpoint
  - Assert row status updates to "retrying"

**Test D — Widget smoke test**
  - Create a minimal HTML page via `page.setContent()` that loads the built widget JS
    inline (read `widget/dist/widget.js` content and inject as a script tag)
  - Call `CRMChatWidget.init({ orgId: 'test', apiUrl: 'http://localhost:8080' })`
  - Assert no JS errors thrown (use `page.on('pageerror')`)
  - Assert `CRMChatWidget` global is defined

### 5. fly.toml

If `fly.toml` exists in the repo root, add the new secrets as commented-out [env] entries:
```toml
# IO Channels — set via: fly secrets set CHAT_JWT_SECRET=... INTERNAL_API_KEY=...
# CHAT_JWT_SECRET = ""       # Required: min 32 chars
# INTERNAL_API_KEY = ""      # Required: internal bridge auth
# LIVEKIT_URL = ""           # Required for voice channel
# LIVEKIT_API_KEY = ""       # Required for voice channel
# LIVEKIT_API_SECRET = ""    # Required for voice channel
# LIVEKIT_WEBHOOK_TOKEN = "" # Required for voice channel
```

Update the fly.toml deploy docs comment or README Deployment section with:
```bash
fly secrets set \
  CHAT_JWT_SECRET=<secret> \
  INTERNAL_API_KEY=<secret> \
  LIVEKIT_URL=<url> \
  LIVEKIT_API_KEY=<key> \
  LIVEKIT_API_SECRET=<secret> \
  LIVEKIT_WEBHOOK_TOKEN=<token>
```

### 6. Final quality gate

Run `task check` in full. It MUST pass completely with:
- All Go tests green, coverage ≥85%
- Widget build succeeds, widget tests green, coverage ≥85%
- Frontend lint, typecheck, tests green, coverage ≥85%
- No uncommitted changes

---

## REQUIREMENTS CHECKLIST

- [ ] Read AGENTS.md and all deft files it references BEFORE writing any code
- [ ] Branch `feat/io-phase7-integration-deploy` created from `feat/io-phase1-channel-gateway`
- [ ] `api/.env.example` updated with all IO vars
- [ ] `docs/env-vars.md` IO section complete and accurate
- [ ] `docker/docker-compose.yml` passes all new vars + adds widget service
- [ ] `.github/workflows/io-channels.yml` created and valid YAML
- [ ] `web/e2e/io-channels.spec.ts` — all 4 test cases implemented
- [ ] `fly.toml` updated (if file exists)
- [ ] `task check` MUST fully pass before creating the PR
- [ ] Conventional commits format for all commits
- [ ] PR targets `feat/io-phase1-channel-gateway`
- [ ] PR title: `chore(io): integration, environment config, and CI/CD (Phase 7)`
