# Environment Variables

Complete reference for all environment variables used by the DEFT Evolution platform.

## API Server (Go)

### Server

- `SERVER_PORT` — HTTP listen port. Default: `8080`.
- `SERVER_HOST` — HTTP listen host. Default: `0.0.0.0`.

### Database

- `SQLITE_PATH` — Path to SQLite database file. Default: `data/deft.db`.

### Authentication (Clerk)

- `CLERK_SECRET_KEY` — Clerk secret key for backend JWT validation. Required in production.
- `CLERK_PUBLISHABLE_KEY` — Clerk publishable key. Required in production.
- `CLERK_ISSUER_URL` — Clerk JWT issuer URL for JWKS validation. Required in production.

### Logging

- `LOG_LEVEL` — Log level: `debug`, `info`, `warn`, `error`. Default: `info`.

### CORS

- `CORS_ORIGINS` — Comma-separated list of allowed origins. Default: `http://localhost:3000`.

### File Uploads

- `UPLOAD_DIR` — Directory for file uploads. Default: `uploads`.
- `UPLOAD_MAX_SIZE` — Maximum upload size in bytes. Default: `104857600` (100MB).

### RBAC

- `RBAC_POLICY_PATH` — Path to RBAC policy YAML. Default: `config/rbac-policy.yaml`.

### OpenTelemetry

- `OTEL_ENABLED` — Enable OpenTelemetry. Default: `false`.
- `OTEL_ENDPOINT` — OTel collector endpoint. Default: empty (disabled).

## Web Frontend (Next.js)

### Public (client-side)

- `NEXT_PUBLIC_API_URL` — Backend API base URL. Example: `https://api.deft.example.com`.
- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY` — Clerk publishable key for frontend auth.

### Server-side

- `CLERK_SECRET_KEY` — Clerk secret key for Next.js middleware JWT validation.

### Build

- `NEXT_TELEMETRY_DISABLED` — Set to `1` to disable Next.js telemetry.

## Docker Compose Overrides

These variables override Docker Compose defaults when set in `.env` alongside `docker-compose.yml`:

- `API_PORT` — Host port for API. Default: `8080`.
- `WEB_PORT` — Host port for web frontend. Default: `3000`.
- `LOG_LEVEL` — Passed through to API container. Default: `info`.
- `CLERK_SECRET_KEY` — Passed through to both containers.
- `CLERK_PUBLISHABLE_KEY` — Passed through to both containers.
- `CLERK_ISSUER_URL` — Passed through to API container.
- `OTEL_ENABLED` — Passed through to API container. Default: `false`.
- `OTEL_ENDPOINT` — Passed through to API container.

## IO Channels

### Chat Gateway

- `CHAT_JWT_SECRET` — HMAC signing secret for chat widget session JWTs. **Required** in production (must be overridden; minimum 32 characters). Default: `change-me-in-production-min-32-chars`. Example: `$(openssl rand -base64 32)`.
- `INTERNAL_API_KEY` — Shared secret for internal bridge authentication between services. **Required** in production (must be overridden). Default: `change-me-in-production`. Example: `$(openssl rand -hex 24)`.

### LiveKit (Voice Channel)

These variables are only required when the voice channel is enabled.

- `LIVEKIT_URL` — WebSocket URL for the LiveKit server. **Required for voice**. No default. Example: `wss://your-livekit-server.example.com`.
- `LIVEKIT_API_KEY` — LiveKit API key for generating access tokens. **Required for voice**. No default. Example: `APIxxxxxxxxxxxxxxx`.
- `LIVEKIT_API_SECRET` — LiveKit API secret for signing access tokens. **Required for voice**. No default. Example: `your-livekit-api-secret`.
- `LIVEKIT_WEBHOOK_TOKEN` — Token for validating inbound LiveKit webhook events. **Required for voice**. No default. Example: `your-livekit-webhook-token`.

### Agentic CLI

The CLI stores credentials client-side in `~/.deft-cli.yaml`. No server-side environment variables are required for the CLI itself.

## Fly.io Secrets

Set via `fly secrets set`:

- `CLERK_SECRET_KEY`
- `CLERK_PUBLISHABLE_KEY`
- `CLERK_ISSUER_URL`
- `CORS_ORIGINS` — Set to production frontend URL.
- `CHAT_JWT_SECRET` — Min 32 chars. Required.
- `INTERNAL_API_KEY` — Required.
- `LIVEKIT_URL` — Required for voice channel.
- `LIVEKIT_API_KEY` — Required for voice channel.
- `LIVEKIT_API_SECRET` — Required for voice channel.
- `LIVEKIT_WEBHOOK_TOKEN` — Required for voice channel.

Set IO secrets via:

```bash
fly secrets set \
  CHAT_JWT_SECRET=<secret> \
  INTERNAL_API_KEY=<secret> \
  LIVEKIT_URL=<url> \
  LIVEKIT_API_KEY=<key> \
  LIVEKIT_API_SECRET=<secret> \
  LIVEKIT_WEBHOOK_TOKEN=<token>
```

## Vercel Environment Variables

Configure in Vercel dashboard:

- `NEXT_PUBLIC_API_URL` — Production API URL (e.g. `https://deft-evolution-api.fly.dev`).
- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`
- `CLERK_SECRET_KEY`
