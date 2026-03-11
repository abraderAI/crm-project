// CRM domain types for sales pipeline, lead scoring, and enrichment.

import type { Thread, Message } from "./api-types";

/** Pipeline stage identifiers matching Go backend pipeline config. */
export type PipelineStage =
  | "new_lead"
  | "contacted"
  | "qualified"
  | "proposal"
  | "negotiation"
  | "closed_won"
  | "closed_lost"
  | "nurturing";

/** Default ordered pipeline stages for the Kanban board. */
export const PIPELINE_STAGES: readonly PipelineStage[] = [
  "new_lead",
  "contacted",
  "qualified",
  "proposal",
  "negotiation",
  "closed_won",
  "closed_lost",
  "nurturing",
] as const;

/** Human-readable labels for each stage. */
export const STAGE_LABELS: Record<PipelineStage, string> = {
  new_lead: "New Lead",
  contacted: "Contacted",
  qualified: "Qualified",
  proposal: "Proposal",
  negotiation: "Negotiation",
  closed_won: "Closed Won",
  closed_lost: "Closed Lost",
  nurturing: "Nurturing",
};

/** Color mappings for pipeline stages (tailwind classes). */
export const STAGE_COLORS: Record<PipelineStage, string> = {
  new_lead: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200",
  contacted: "bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200",
  qualified: "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200",
  proposal: "bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200",
  negotiation: "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200",
  closed_won: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
  closed_lost: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200",
  nurturing: "bg-teal-100 text-teal-800 dark:bg-teal-900 dark:text-teal-200",
};

/** Parsed lead metadata extracted from a thread's metadata JSON. */
export interface LeadData {
  company?: string;
  value?: number;
  assigned_to?: string;
  score?: number;
  contact_name?: string;
  contact_email?: string;
  source?: string;
  customer_org_id?: string;
}

/** A lead card represents a thread in the CRM pipeline context. */
export interface LeadCard {
  thread: Thread;
  lead: LeadData;
  stage: PipelineStage;
}

/** A single scoring rule contribution. */
export interface ScoreRule {
  name: string;
  description: string;
  points: number;
  matched: boolean;
}

/** Full score breakdown returned by the scoring endpoint. */
export interface ScoreBreakdown {
  total: number;
  rules: ScoreRule[];
}

/** AI enrichment data returned by the enrich endpoint. */
export interface EnrichmentData {
  summary?: string;
  next_action?: string;
  enriched_at?: string;
}

/** Pipeline dashboard statistics. */
export interface PipelineStats {
  total_leads: number;
  total_value: number;
  stage_counts: Record<string, number>;
  conversion_rate: number;
  average_value: number;
}

/** Extract LeadData from a thread's raw metadata string. */
export function parseLeadData(metadata: string | Record<string, unknown>): LeadData {
  let parsed: Record<string, unknown>;
  if (typeof metadata === "string") {
    try {
      const result: unknown = JSON.parse(metadata);
      if (typeof result === "object" && result !== null && !Array.isArray(result)) {
        parsed = result as Record<string, unknown>;
      } else {
        return {};
      }
    } catch {
      return {};
    }
  } else {
    parsed = metadata;
  }

  return {
    company: typeof parsed.company === "string" ? parsed.company : undefined,
    value: typeof parsed.value === "number" ? parsed.value : undefined,
    assigned_to: typeof parsed.assigned_to === "string" ? parsed.assigned_to : undefined,
    score: typeof parsed.score === "number" ? parsed.score : undefined,
    contact_name: typeof parsed.contact_name === "string" ? parsed.contact_name : undefined,
    contact_email: typeof parsed.contact_email === "string" ? parsed.contact_email : undefined,
    source: typeof parsed.source === "string" ? parsed.source : undefined,
    customer_org_id:
      typeof parsed.customer_org_id === "string" ? parsed.customer_org_id : undefined,
  };
}

/** Determine the pipeline stage from thread metadata/generated columns. */
export function resolveStage(thread: Thread): PipelineStage {
  const stage = thread.stage;
  if (stage && PIPELINE_STAGES.includes(stage as PipelineStage)) {
    return stage as PipelineStage;
  }
  return "new_lead";
}

/** Convert threads into LeadCards grouped by pipeline stage. */
export function threadsToLeadCards(threads: Thread[]): LeadCard[] {
  return threads.map((thread) => ({
    thread,
    lead: parseLeadData(thread.metadata),
    stage: resolveStage(thread),
  }));
}

/** Group lead cards by pipeline stage. */
export function groupByStage(cards: LeadCard[]): Record<PipelineStage, LeadCard[]> {
  const grouped: Record<PipelineStage, LeadCard[]> = {
    new_lead: [],
    contacted: [],
    qualified: [],
    proposal: [],
    negotiation: [],
    closed_won: [],
    closed_lost: [],
    nurturing: [],
  };
  for (const card of cards) {
    grouped[card.stage].push(card);
  }
  return grouped;
}

/** Format currency value for display. */
export function formatCurrency(value: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value);
}

/** Compute pipeline stats from a collection of lead cards. */
export function computePipelineStats(cards: LeadCard[]): PipelineStats {
  const stageCounts: Record<string, number> = {};
  let totalValue = 0;
  let closedWon = 0;
  let closedTotal = 0;

  for (const card of cards) {
    stageCounts[card.stage] = (stageCounts[card.stage] ?? 0) + 1;
    totalValue += card.lead.value ?? 0;
    if (card.stage === "closed_won") closedWon++;
    if (card.stage === "closed_won" || card.stage === "closed_lost") closedTotal++;
  }

  return {
    total_leads: cards.length,
    total_value: totalValue,
    stage_counts: stageCounts,
    conversion_rate: closedTotal > 0 ? (closedWon / closedTotal) * 100 : 0,
    average_value: cards.length > 0 ? totalValue / cards.length : 0,
  };
}

/** Extract enrichment data from thread metadata. */
export function parseEnrichmentData(
  metadata: string | Record<string, unknown>,
): EnrichmentData | null {
  let parsed: Record<string, unknown>;
  if (typeof metadata === "string") {
    try {
      const result: unknown = JSON.parse(metadata);
      if (typeof result === "object" && result !== null && !Array.isArray(result)) {
        parsed = result as Record<string, unknown>;
      } else {
        return null;
      }
    } catch {
      return null;
    }
  } else {
    parsed = metadata;
  }

  const enrichment = parsed.enrichment;
  if (typeof enrichment !== "object" || enrichment === null || Array.isArray(enrichment)) {
    return null;
  }

  const e = enrichment as Record<string, unknown>;
  return {
    summary: typeof e.summary === "string" ? e.summary : undefined,
    next_action: typeof e.next_action === "string" ? e.next_action : undefined,
    enriched_at: typeof e.enriched_at === "string" ? e.enriched_at : undefined,
  };
}

/** Parse score breakdown from metadata or direct API response. */
export function parseScoreBreakdown(data: unknown): ScoreBreakdown | null {
  if (typeof data !== "object" || data === null || Array.isArray(data)) return null;
  const obj = data as Record<string, unknown>;
  if (typeof obj.total !== "number") return null;
  if (!Array.isArray(obj.rules)) return null;

  const rules: ScoreRule[] = [];
  for (const r of obj.rules) {
    if (typeof r !== "object" || r === null || Array.isArray(r)) continue;
    const rule = r as Record<string, unknown>;
    if (
      typeof rule.name === "string" &&
      typeof rule.description === "string" &&
      typeof rule.points === "number" &&
      typeof rule.matched === "boolean"
    ) {
      rules.push({
        name: rule.name,
        description: rule.description,
        points: rule.points,
        matched: rule.matched,
      });
    }
  }

  return { total: obj.total, rules };
}

/** Filter lead cards by an optional assignee filter. */
export function filterLeadsByAssignee(cards: LeadCard[], assignee: string): LeadCard[] {
  if (!assignee || assignee === "all") return cards;
  return cards.filter((c) => c.lead.assigned_to === assignee);
}

/** Filter lead cards by minimum score. */
export function filterLeadsByMinScore(cards: LeadCard[], minScore: number): LeadCard[] {
  if (minScore <= 0) return cards;
  return cards.filter((c) => (c.lead.score ?? 0) >= minScore);
}

/** Get unique assignees from lead cards. */
export function getUniqueAssignees(cards: LeadCard[]): string[] {
  const set = new Set<string>();
  for (const card of cards) {
    if (card.lead.assigned_to) {
      set.add(card.lead.assigned_to);
    }
  }
  return Array.from(set).sort();
}

/** Type guard for activity message types relevant to CRM. */
export function isCrmActivityMessage(message: Message): boolean {
  return ["note", "email", "call_log", "system"].includes(message.type);
}
