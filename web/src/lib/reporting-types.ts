// Reporting types matching Go backend models in api/internal/reporting/.

/** Query parameters accepted by all reporting endpoints. */
export interface ReportParams {
  from?: string; // "YYYY-MM-DD"
  to?: string; // "YYYY-MM-DD"
  assignee?: string;
}

/** Single day volume count. */
export interface DailyCount {
  date: string;
  count: number;
}

/** Per-assignee count. */
export interface AssigneeCount {
  user_id: string;
  name: string;
  count: number;
}

/** Pipeline stage count. */
export interface StageCount {
  stage: string;
  count: number;
}

/** Score-distribution bucket. */
export interface BucketCount {
  range: string;
  count: number;
}

/** Stage-to-stage conversion rate. */
export interface StageConversion {
  from_stage: string;
  to_stage: string;
  rate: number;
}

/** Average time in a pipeline stage. */
export interface StageAvgTime {
  stage: string;
  avg_hours: number | null;
}

/** Org-scoped support report metrics. */
export interface SupportMetrics {
  status_breakdown: Record<string, number>;
  volume_over_time: DailyCount[];
  avg_resolution_hours: number | null;
  tickets_by_assignee: AssigneeCount[];
  tickets_by_priority: Record<string, number>;
  avg_first_response_hours: number | null;
  overdue_count: number;
}

/** Org-scoped sales report metrics. */
export interface SalesMetrics {
  pipeline_funnel: StageCount[];
  lead_velocity: DailyCount[];
  win_rate: number;
  loss_rate: number;
  avg_deal_value: number | null;
  leads_by_assignee: AssigneeCount[];
  score_distribution: BucketCount[];
  stage_conversion_rates: StageConversion[];
  avg_time_in_stage: StageAvgTime[];
}

/** Per-org support summary in admin report. */
export interface OrgSupportSummary {
  org_id: string;
  org_name: string;
  org_slug: string;
  open_count: number;
  overdue_count: number;
  avg_resolution_hours: number | null;
  avg_first_response_hours: number | null;
  total_in_range: number;
}

/** Per-org sales summary in admin report. */
export interface OrgSalesSummary {
  org_id: string;
  org_name: string;
  org_slug: string;
  total_leads: number;
  win_rate: number;
  avg_deal_value: number | null;
  open_pipeline_count: number;
}

/** Platform-admin support report (extends org-scoped with org breakdown). */
export interface AdminSupportMetrics extends SupportMetrics {
  org_breakdown: OrgSupportSummary[];
}

/** Platform-admin sales report (extends org-scoped with org breakdown). */
export interface AdminSalesMetrics extends SalesMetrics {
  org_breakdown: OrgSalesSummary[];
}
