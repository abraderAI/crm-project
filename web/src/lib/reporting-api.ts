import type { ReportParams } from "./reporting-types";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

/** Build a query string from ReportParams for export URLs. */
function buildQueryString(params: ReportParams): string {
  const record: Record<string, string> = {};
  if (params.from) record["from"] = params.from;
  if (params.to) record["to"] = params.to;
  if (params.assignee) record["assignee"] = params.assignee;
  const entries = Object.entries(record);
  if (entries.length === 0) return "";
  return `?${new URLSearchParams(entries).toString()}`;
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
