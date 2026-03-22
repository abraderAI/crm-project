import type { DeftDepartment, Tier, WidgetConfig } from "./tier-types";

/** Well-known widget IDs used across all tiers. */
export const WIDGET_IDS = {
  // Tier 1 (Anonymous)
  DOCS_HIGHLIGHTS: "docs-highlights",
  FORUM_HIGHLIGHTS: "forum-highlights",
  GET_STARTED: "get-started",

  // Tier 2 (Registered Developer)
  MY_PROFILE: "my-profile",
  MY_FORUM_ACTIVITY: "my-forum-activity",
  MY_SUPPORT_TICKETS: "my-support-tickets",
  UPGRADE_CTA: "upgrade-cta",

  // Tier 3 (Paying Customer)
  ORG_OVERVIEW: "org-overview",
  ORG_SUPPORT_TICKETS: "org-support-tickets",
  BILLING_STATUS: "billing-status",
  ORG_SUPPORT_DASHBOARD: "org-support-dashboard",

  // Tier 4 (DEFT Employee)
  LEAD_PIPELINE: "lead-pipeline",
  RECENT_LEADS: "recent-leads",
  CONVERSION_METRICS: "conversion-metrics",
  TICKET_QUEUE: "ticket-queue",
  TICKET_STATS: "ticket-stats",
  BILLING_OVERVIEW: "billing-overview",

  // Tier 5 (Customer Org Admin)
  ORG_ACCESS_CONTROL: "org-access-control",
  ORG_RBAC_EDITOR: "org-rbac-editor",

  // Tier 6 (Platform Admin)
  SYSTEM_HEALTH: "system-health",
  RECENT_AUDIT_LOG: "recent-audit-log",
} as const;

/** Set of all valid widget IDs for validation. */
export const VALID_WIDGET_IDS: ReadonlySet<string> = new Set(Object.values(WIDGET_IDS));

/** Helper to create a visible widget config. */
function w(widgetId: string): WidgetConfig {
  return { widget_id: widgetId, visible: true };
}

/** Default layout for Tier 1: Anonymous visitors. */
const TIER_1_DEFAULT: WidgetConfig[] = [
  w(WIDGET_IDS.DOCS_HIGHLIGHTS),
  w(WIDGET_IDS.FORUM_HIGHLIGHTS),
  w(WIDGET_IDS.GET_STARTED),
];

/** Default layout for Tier 2: Registered developers. */
const TIER_2_DEFAULT: WidgetConfig[] = [
  w(WIDGET_IDS.MY_PROFILE),
  w(WIDGET_IDS.MY_FORUM_ACTIVITY),
  w(WIDGET_IDS.MY_SUPPORT_TICKETS),
  w(WIDGET_IDS.UPGRADE_CTA),
];

/** Default layout for Tier 3: Paying customers (member variant). */
const TIER_3_DEFAULT: WidgetConfig[] = [
  w(WIDGET_IDS.ORG_OVERVIEW),
  w(WIDGET_IDS.ORG_SUPPORT_TICKETS),
  w(WIDGET_IDS.MY_FORUM_ACTIVITY),
];

/** Default layout for Tier 4 — Sales department. */
const TIER_4_SALES_DEFAULT: WidgetConfig[] = [
  w(WIDGET_IDS.LEAD_PIPELINE),
  w(WIDGET_IDS.RECENT_LEADS),
  w(WIDGET_IDS.CONVERSION_METRICS),
];

/** Default layout for Tier 4 — Support department. */
const TIER_4_SUPPORT_DEFAULT: WidgetConfig[] = [
  w(WIDGET_IDS.TICKET_QUEUE),
  w(WIDGET_IDS.TICKET_STATS),
];

/** Default layout for Tier 4 — Finance department. */
const TIER_4_FINANCE_DEFAULT: WidgetConfig[] = [w(WIDGET_IDS.BILLING_OVERVIEW)];

/** Default layout for Tier 5: Customer org admins. */
const TIER_5_DEFAULT: WidgetConfig[] = [
  w(WIDGET_IDS.ORG_ACCESS_CONTROL),
  w(WIDGET_IDS.ORG_RBAC_EDITOR),
  w(WIDGET_IDS.ORG_SUPPORT_DASHBOARD),
  w(WIDGET_IDS.BILLING_STATUS),
];

/** Default layout for Tier 6: Platform admins (widgets not used — admin console has real data). */
const TIER_6_DEFAULT: WidgetConfig[] = [];

/** Map from tier number to base default layout. */
const TIER_DEFAULTS: Record<Tier, WidgetConfig[]> = {
  1: TIER_1_DEFAULT,
  2: TIER_2_DEFAULT,
  3: TIER_3_DEFAULT,
  4: TIER_4_SALES_DEFAULT,
  5: TIER_5_DEFAULT,
  6: TIER_6_DEFAULT,
};

/** Map from DEFT department to Tier 4 sub-layout. */
const TIER_4_DEPARTMENT_DEFAULTS: Record<DeftDepartment, WidgetConfig[]> = {
  sales: TIER_4_SALES_DEFAULT,
  support: TIER_4_SUPPORT_DEFAULT,
  finance: TIER_4_FINANCE_DEFAULT,
};

/**
 * Get the default widget layout for a given tier and optional department.
 * For Tier 4, the department determines which department-specific layout is returned.
 */
export function getDefaultLayout(tier: Tier, department?: DeftDepartment | null): WidgetConfig[] {
  if (tier === 4 && department) {
    return [...(TIER_4_DEPARTMENT_DEFAULTS[department] ?? TIER_4_SALES_DEFAULT)];
  }
  return [...(TIER_DEFAULTS[tier] ?? TIER_1_DEFAULT)];
}

/** Validate that a widget layout contains only known widget IDs. */
export function validateLayout(layout: WidgetConfig[]): string[] {
  const errors: string[] = [];
  const seen = new Set<string>();

  for (const item of layout) {
    if (!VALID_WIDGET_IDS.has(item.widget_id)) {
      errors.push(`Unknown widget ID: ${item.widget_id}`);
    }
    if (seen.has(item.widget_id)) {
      errors.push(`Duplicate widget ID: ${item.widget_id}`);
    }
    seen.add(item.widget_id);
  }

  return errors;
}
