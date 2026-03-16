import { buildHeaders, buildUrl, parseResponse } from "./api-client";

/** Response from POST /v1/me/upgrade. */
export interface UpgradeResponse {
  tier: number;
  message: string;
}

/**
 * Upgrade the current user from Tier 2 (Developer) to Tier 3 (Customer).
 * Calls POST /v1/me/upgrade with the user's auth token.
 */
export async function upgradeToCustomer(token: string): Promise<UpgradeResponse> {
  const url = buildUrl("/me/upgrade");
  const response = await fetch(url, {
    method: "POST",
    headers: buildHeaders(token),
  });
  return parseResponse<UpgradeResponse>(response);
}
