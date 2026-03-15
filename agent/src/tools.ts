/**
 * Tool functions for the voice agent to interact with CRM data
 * via the internal bridge API.
 */

import type { AgentConfig } from "./config.js";

/** ContactResult represents a matching contact thread from the CRM. */
export interface ContactResult {
  id: string;
  title: string;
  metadata: string;
}

/** ThreadSummary represents a brief summary of a CRM thread. */
export interface ThreadSummary {
  id: string;
  title: string;
  body: string;
  metadata: string;
  message_count: number;
  created_at: string;
}

/** FetchFn abstracts the fetch function for testability. */
export type FetchFn = (url: string, init?: RequestInit) => Promise<Response>;

/** Default fetch function using global fetch. */
const defaultFetch: FetchFn = globalThis.fetch?.bind(globalThis) ?? (async () => {
  throw new Error("fetch is not available");
});

/**
 * Creates a set of CRM tool functions bound to the given config.
 */
export function createTools(config: AgentConfig, fetchFn: FetchFn = defaultFetch) {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (config.internalApiKey) {
    headers["X-Internal-Key"] = config.internalApiKey;
  }

  return {
    /**
     * Looks up contacts in the CRM by email or phone number.
     * Returns matching thread records.
     */
    async lookupContact(params: { email?: string; phone?: string }): Promise<ContactResult[]> {
      const query = new URLSearchParams();
      if (params.email) query.set("email", params.email);
      if (params.phone) query.set("phone", params.phone);

      const url = `${config.crmBaseUrl}/v1/internal/contacts/lookup?${query.toString()}`;
      const resp = await fetchFn(url, { headers });

      if (!resp.ok) {
        throw new Error(`Contact lookup failed: ${resp.status}`);
      }

      const data = (await resp.json()) as { contacts: ContactResult[] };
      return data.contacts;
    },

    /**
     * Gets a summary of a specific CRM thread.
     */
    async getThreadSummary(threadId: string): Promise<ThreadSummary> {
      const url = `${config.crmBaseUrl}/v1/internal/threads/${threadId}/summary`;
      const resp = await fetchFn(url, { headers });

      if (!resp.ok) {
        throw new Error(`Thread summary failed: ${resp.status}`);
      }

      return (await resp.json()) as ThreadSummary;
    },
  };
}
