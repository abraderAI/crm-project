import type { ApiKey, ApiKeyCreateResponse } from "./api-types";
import { buildHeaders, buildUrl, parseResponse } from "./api-client";

/**
 * Fetch the current user's API keys.
 * Requires authentication token.
 */
export async function fetchApiKeys(token: string, org = "default"): Promise<ApiKey[]> {
  const url = buildUrl(`/orgs/${org}/api-keys`);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  const result = await parseResponse<{ data: ApiKey[] }>(response);
  return result.data;
}

/**
 * Create a new API key. The full key is returned exactly once in the response.
 * Requires authentication token.
 */
export async function createApiKey(
  token: string,
  name: string,
  org = "default",
): Promise<ApiKeyCreateResponse> {
  const url = buildUrl(`/orgs/${org}/api-keys`);
  const response = await fetch(url, {
    method: "POST",
    headers: buildHeaders(token),
    body: JSON.stringify({ name }),
  });
  return parseResponse<ApiKeyCreateResponse>(response);
}

/**
 * Revoke (delete) an API key by ID.
 * Requires authentication token.
 */
export async function revokeApiKey(token: string, keyId: string, org = "default"): Promise<void> {
  const url = buildUrl(`/orgs/${org}/api-keys/${keyId}`);
  const response = await fetch(url, {
    method: "DELETE",
    headers: buildHeaders(token),
  });
  if (!response.ok) {
    await parseResponse<never>(response);
  }
}
