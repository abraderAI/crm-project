import { auth } from "@clerk/nextjs/server";

import type {
  AdminSalesMetrics,
  AdminSupportMetrics,
  ReportParams,
  SalesMetrics,
  SupportMetrics,
} from "./reporting-types";
import { buildHeaders, buildUrl, parseResponse } from "./api-client";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

/** Get a Clerk JWT token for server-side requests. Throws if unauthenticated. */
async function getToken(): Promise<string> {
  const { getToken: clerkGetToken } = await auth();
  const token = await clerkGetToken();
  if (!token) {
    throw new Error("Unauthenticated");
  }
  return token;
}

/** Build query string from ReportParams, omitting empty values. */
function buildParamsRecord(params: ReportParams): Record<string, string> {
  const record: Record<string, string> = {};
  if (params.from) record["from"] = params.from;
  if (params.to) record["to"] = params.to;
  if (params.assignee) record["assignee"] = params.assignee;
  return record;
}

/** Build a query string from ReportParams for export URLs. */
function buildQueryString(params: ReportParams): string {
  const entries = Object.entries(buildParamsRecord(params));
  if (entries.length === 0) return "";
  const searchParams = new URLSearchParams(entries);
  return `?${searchParams.toString()}`;
}

/** Server-side fetch with query params for report endpoints. */
async function reportFetch<T>(path: string, params: ReportParams): Promise<T> {
  const token = await getToken();
  const url = buildUrl(path, buildParamsRecord(params));
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<T>(response);
}

// --- Org-scoped report fetchers ---

/** Fetch org-scoped support metrics. */
export async function getSupportMetrics(
  orgId: string,
  params: ReportParams,
): Promise<SupportMetrics> {
  return reportFetch<SupportMetrics>(`/orgs/${orgId}/reports/support`, params);
}

/** Fetch org-scoped sales metrics. */
export async function getSalesMetrics(orgId: string, params: ReportParams): Promise<SalesMetrics> {
  return reportFetch<SalesMetrics>(`/orgs/${orgId}/reports/sales`, params);
}

// --- Admin report fetchers ---

/** Fetch platform-admin support metrics. */
export async function getAdminSupportMetrics(params: ReportParams): Promise<AdminSupportMetrics> {
  return reportFetch<AdminSupportMetrics>("/admin/reports/support", params);
}

/** Fetch platform-admin sales metrics. */
export async function getAdminSalesMetrics(params: ReportParams): Promise<AdminSalesMetrics> {
  return reportFetch<AdminSalesMetrics>("/admin/reports/sales", params);
}

// --- Export URL builders ---

/** Build the full URL for org-scoped support CSV export. */
export function getSupportExportUrl(orgId: string, params: ReportParams): string {
  return `${API_BASE_URL}/v1/orgs/${orgId}/reports/support/export${buildQueryString(params)}`;
}

/** Build the full URL for org-scoped sales CSV export. */
export function getSalesExportUrl(orgId: string, params: ReportParams): string {
  return `${API_BASE_URL}/v1/orgs/${orgId}/reports/sales/export${buildQueryString(params)}`;
}

/** Build the full URL for admin support CSV export. */
export function getAdminSupportExportUrl(params: ReportParams): string {
  return `${API_BASE_URL}/v1/admin/reports/support/export${buildQueryString(params)}`;
}

/** Build the full URL for admin sales CSV export. */
export function getAdminSalesExportUrl(params: ReportParams): string {
  return `${API_BASE_URL}/v1/admin/reports/sales/export${buildQueryString(params)}`;
}
