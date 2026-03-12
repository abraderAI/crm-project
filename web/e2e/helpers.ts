import { type APIRequestContext } from "@playwright/test";

/** Base URL for the API server. */
export const API_URL = process.env.API_URL ?? "http://localhost:8080";

/** Timestamp-based unique suffix for test isolation. */
export function uniqueSuffix(): string {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 6);
}

/** Create an org via the API and return its data. */
export async function createOrg(
  request: APIRequestContext,
  name?: string,
): Promise<Record<string, unknown>> {
  const suffix = uniqueSuffix();
  const orgName = name ?? `Smoke Org ${suffix}`;
  const resp = await request.post(`${API_URL}/v1/orgs`, {
    data: { name: orgName, description: "E2E smoke test org" },
    headers: { "Content-Type": "application/json" },
  });
  return (await resp.json()) as Record<string, unknown>;
}

/** Create a space under an org. */
export async function createSpace(
  request: APIRequestContext,
  orgId: string,
  name?: string,
): Promise<Record<string, unknown>> {
  const suffix = uniqueSuffix();
  const resp = await request.post(`${API_URL}/v1/orgs/${orgId}/spaces`, {
    data: {
      name: name ?? `Smoke Space ${suffix}`,
      description: "E2E space",
      type: "general",
    },
    headers: { "Content-Type": "application/json" },
  });
  return (await resp.json()) as Record<string, unknown>;
}

/** Create a board under a space. */
export async function createBoard(
  request: APIRequestContext,
  orgId: string,
  spaceId: string,
  name?: string,
): Promise<Record<string, unknown>> {
  const suffix = uniqueSuffix();
  const resp = await request.post(
    `${API_URL}/v1/orgs/${orgId}/spaces/${spaceId}/boards`,
    {
      data: { name: name ?? `Smoke Board ${suffix}`, description: "E2E board" },
      headers: { "Content-Type": "application/json" },
    },
  );
  return (await resp.json()) as Record<string, unknown>;
}

/** Create a thread under a board. */
export async function createThread(
  request: APIRequestContext,
  orgId: string,
  spaceId: string,
  boardId: string,
  title?: string,
  metadata?: Record<string, unknown>,
): Promise<Record<string, unknown>> {
  const suffix = uniqueSuffix();
  const resp = await request.post(
    `${API_URL}/v1/orgs/${orgId}/spaces/${spaceId}/boards/${boardId}/threads`,
    {
      data: {
        title: title ?? `Smoke Thread ${suffix}`,
        body: "E2E test thread body",
        ...(metadata ? { metadata } : {}),
      },
      headers: { "Content-Type": "application/json" },
    },
  );
  return (await resp.json()) as Record<string, unknown>;
}

/** Create a message under a thread. */
export async function createMessage(
  request: APIRequestContext,
  orgId: string,
  spaceId: string,
  boardId: string,
  threadId: string,
  body?: string,
): Promise<Record<string, unknown>> {
  const resp = await request.post(
    `${API_URL}/v1/orgs/${orgId}/spaces/${spaceId}/boards/${boardId}/threads/${threadId}/messages`,
    {
      data: {
        body: body ?? "E2E smoke test message",
        type: "comment",
      },
      headers: { "Content-Type": "application/json" },
    },
  );
  return (await resp.json()) as Record<string, unknown>;
}

/** Measure response time of a request (in ms). */
export async function measureResponseTime(
  request: APIRequestContext,
  url: string,
): Promise<{ status: number; timeMs: number }> {
  const start = Date.now();
  const resp = await request.get(url);
  const timeMs = Date.now() - start;
  return { status: resp.status(), timeMs };
}
