/** User tier levels matching backend TierResolver output. */
export type Tier = 1 | 2 | 3 | 4 | 5 | 6;

/** DEFT employee department sub-types for Tier 4. */
export type DeftDepartment = "sales" | "support" | "finance";

/** Tier sub-type: org owner flag or DEFT department. */
export type TierSubType = "owner" | DeftDepartment | null;

/** Tier information returned by GET /api/me/tier. */
export interface TierInfo {
  tier: Tier;
  sub_type: TierSubType;
  org_id?: string | null;
  deft_department?: DeftDepartment | null;
}

/** A single widget configuration in a home layout. */
export interface WidgetConfig {
  widget_id: string;
  visible: boolean;
}

/** User home preferences stored server-side. */
export interface HomePreferences {
  user_id: string;
  tier: Tier;
  layout: WidgetConfig[];
}

/** Tier display labels for UI. */
export const TIER_LABELS: Record<Tier, string> = {
  1: "Anonymous",
  2: "Registered Developer",
  3: "Paying Customer",
  4: "DEFT Employee",
  5: "Customer Org Admin",
  6: "Platform Admin",
};
