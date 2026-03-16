import { buildHeaders, buildUrl, parseResponse } from "./api-client";
import type { HomePreferences, TierInfo, WidgetConfig } from "./tier-types";

/**
 * Fetch the current user's tier information.
 * Anonymous users (no token) are returned as Tier 1.
 */
export async function fetchTierInfo(token?: string | null): Promise<TierInfo> {
  if (!token) {
    return { tier: 1, sub_type: null };
  }
  const url = buildUrl("/me/tier");
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<TierInfo>(response);
}

/**
 * Fetch the current user's home layout preferences.
 * Returns null if no preferences are saved (frontend uses default layout).
 */
export async function fetchHomePreferences(token: string): Promise<HomePreferences | null> {
  const url = buildUrl("/me/home-preferences");
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });

  if (response.status === 404) {
    return null;
  }

  return parseResponse<HomePreferences>(response);
}

/**
 * Save the current user's home layout preferences.
 * Validates widget IDs on the server; returns 400 for invalid layouts.
 */
export async function saveHomePreferences(
  token: string,
  layout: WidgetConfig[],
): Promise<HomePreferences> {
  const url = buildUrl("/me/home-preferences");
  const response = await fetch(url, {
    method: "PUT",
    headers: buildHeaders(token),
    body: JSON.stringify({ layout }),
  });
  return parseResponse<HomePreferences>(response);
}
