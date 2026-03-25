import { describe, expect, it } from "vitest";
import type { Thread, Message } from "./api-types";
import {
  PIPELINE_STAGES,
  STAGE_LABELS,
  STAGE_COLORS,
  STAGE_PROBABILITIES,
  parseLeadData,
  resolveStage,
  threadsToLeadCards,
  groupByStage,
  formatCurrency,
  computePipelineStats,
  parseEnrichmentData,
  parseScoreBreakdown,
  filterLeadsByAssignee,
  filterLeadsByMinScore,
  getUniqueAssignees,
  isCrmActivityMessage,
  calculateWeightedForecast,
  getEffectiveProbability,
  isOverdue,
  daysBetween,
  OPPORTUNITY_TYPES,
  OPPORTUNITY_TYPE_LABELS,
  LEAD_SOURCES,
  LEAD_SOURCE_LABELS,
  COMPANY_STATUSES,
  COMPANY_STATUS_LABELS,
} from "./crm-types";

function makeThread(overrides: Partial<Thread> = {}): Thread {
  return {
    id: "t-1",
    board_id: "b-1",
    title: "Lead: Acme",
    slug: "lead-acme",
    metadata: '{"company":"Acme","value":50000,"assigned_to":"alice","score":75}',
    author_id: "u-1",
    is_pinned: false,
    is_locked: false,
    is_hidden: false,
    vote_score: 0,
    stage: "new_lead",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
    ...overrides,
  };
}

describe("PIPELINE_STAGES", () => {
  it("has 8 stages", () => {
    expect(PIPELINE_STAGES).toHaveLength(8);
  });

  it("starts with new_lead", () => {
    expect(PIPELINE_STAGES[0]).toBe("new_lead");
  });

  it("has labels for all stages", () => {
    for (const stage of PIPELINE_STAGES) {
      expect(STAGE_LABELS[stage]).toBeDefined();
    }
  });

  it("has colors for all stages", () => {
    for (const stage of PIPELINE_STAGES) {
      expect(STAGE_COLORS[stage]).toBeDefined();
    }
  });
});

describe("parseLeadData", () => {
  it("parses lead data from JSON string", () => {
    const result = parseLeadData('{"company":"Acme","value":50000}');
    expect(result.company).toBe("Acme");
    expect(result.value).toBe(50000);
  });

  it("parses lead data from object", () => {
    const result = parseLeadData({ company: "Acme", score: 80 });
    expect(result.company).toBe("Acme");
    expect(result.score).toBe(80);
  });

  it("returns empty for invalid JSON", () => {
    expect(parseLeadData("not json")).toEqual({});
  });

  it("returns empty for empty string", () => {
    expect(parseLeadData("")).toEqual({});
  });

  it("returns empty for JSON array", () => {
    expect(parseLeadData("[1,2]")).toEqual({});
  });

  it("returns empty for JSON number", () => {
    expect(parseLeadData("42")).toEqual({});
  });

  it("extracts all known fields", () => {
    const data = parseLeadData({
      company: "Acme",
      value: 100000,
      assigned_to: "alice",
      score: 90,
      contact_name: "Bob",
      contact_email: "bob@acme.com",
      source: "website",
      customer_org_id: "org-123",
    });
    expect(data.company).toBe("Acme");
    expect(data.value).toBe(100000);
    expect(data.assigned_to).toBe("alice");
    expect(data.score).toBe(90);
    expect(data.contact_name).toBe("Bob");
    expect(data.contact_email).toBe("bob@acme.com");
    expect(data.source).toBe("website");
    expect(data.customer_org_id).toBe("org-123");
  });

  it("ignores fields with wrong types", () => {
    const data = parseLeadData({ company: 123, value: "not-a-number" });
    expect(data.company).toBeUndefined();
    expect(data.value).toBeUndefined();
  });
});

describe("resolveStage", () => {
  it("returns thread stage when valid", () => {
    expect(resolveStage(makeThread({ stage: "qualified" }))).toBe("qualified");
  });

  it("returns new_lead for undefined stage", () => {
    expect(resolveStage(makeThread({ stage: undefined }))).toBe("new_lead");
  });

  it("returns new_lead for unknown stage", () => {
    expect(resolveStage(makeThread({ stage: "unknown_stage" }))).toBe("new_lead");
  });

  it("returns new_lead for empty string stage", () => {
    expect(resolveStage(makeThread({ stage: "" }))).toBe("new_lead");
  });
});

describe("threadsToLeadCards", () => {
  it("converts threads to lead cards", () => {
    const threads = [makeThread(), makeThread({ id: "t-2", stage: "proposal" })];
    const cards = threadsToLeadCards(threads);
    expect(cards).toHaveLength(2);
    expect(cards[0]!.stage).toBe("new_lead");
    expect(cards[1]!.stage).toBe("proposal");
  });

  it("parses lead data from metadata", () => {
    const cards = threadsToLeadCards([makeThread()]);
    expect(cards[0]!.lead.company).toBe("Acme");
    expect(cards[0]!.lead.value).toBe(50000);
  });

  it("handles empty array", () => {
    expect(threadsToLeadCards([])).toEqual([]);
  });
});

describe("groupByStage", () => {
  it("groups cards by stage", () => {
    const cards = threadsToLeadCards([
      makeThread({ id: "t-1", stage: "new_lead" }),
      makeThread({ id: "t-2", stage: "new_lead" }),
      makeThread({ id: "t-3", stage: "qualified" }),
    ]);
    const grouped = groupByStage(cards);
    expect(grouped.new_lead).toHaveLength(2);
    expect(grouped.qualified).toHaveLength(1);
    expect(grouped.proposal).toHaveLength(0);
  });

  it("returns empty arrays for all stages with no cards", () => {
    const grouped = groupByStage([]);
    expect(Object.keys(grouped)).toHaveLength(8);
    for (const stage of PIPELINE_STAGES) {
      expect(grouped[stage]).toEqual([]);
    }
  });
});

describe("formatCurrency", () => {
  it("formats positive values", () => {
    expect(formatCurrency(50000)).toBe("$50,000");
  });

  it("formats zero", () => {
    expect(formatCurrency(0)).toBe("$0");
  });

  it("formats large values", () => {
    expect(formatCurrency(1234567)).toBe("$1,234,567");
  });
});

describe("computePipelineStats", () => {
  it("computes stats for empty array", () => {
    const stats = computePipelineStats([]);
    expect(stats.total_leads).toBe(0);
    expect(stats.total_value).toBe(0);
    expect(stats.conversion_rate).toBe(0);
    expect(stats.average_value).toBe(0);
  });

  it("computes stats with leads", () => {
    const cards = threadsToLeadCards([
      makeThread({ id: "t-1", stage: "new_lead", metadata: '{"value":10000}' }),
      makeThread({ id: "t-2", stage: "closed_won", metadata: '{"value":20000}' }),
      makeThread({ id: "t-3", stage: "closed_lost", metadata: '{"value":30000}' }),
    ]);
    const stats = computePipelineStats(cards);
    expect(stats.total_leads).toBe(3);
    expect(stats.total_value).toBe(60000);
    expect(stats.conversion_rate).toBe(50);
    expect(stats.average_value).toBe(20000);
    expect(stats.stage_counts["new_lead"]).toBe(1);
    expect(stats.stage_counts["closed_won"]).toBe(1);
    expect(stats.stage_counts["closed_lost"]).toBe(1);
  });

  it("handles leads without value", () => {
    const cards = threadsToLeadCards([makeThread({ metadata: "{}" })]);
    const stats = computePipelineStats(cards);
    expect(stats.total_value).toBe(0);
    expect(stats.average_value).toBe(0);
  });
});

describe("parseEnrichmentData", () => {
  it("parses enrichment from JSON string with enrichment key", () => {
    const meta = JSON.stringify({
      enrichment: { summary: "Good lead", next_action: "Follow up", enriched_at: "2025-01-01" },
    });
    const result = parseEnrichmentData(meta);
    expect(result).toEqual({
      summary: "Good lead",
      next_action: "Follow up",
      enriched_at: "2025-01-01",
    });
  });

  it("parses enrichment from object", () => {
    const result = parseEnrichmentData({
      enrichment: { summary: "Promising", next_action: "Send proposal" },
    });
    expect(result?.summary).toBe("Promising");
  });

  it("returns null when no enrichment key", () => {
    expect(parseEnrichmentData({ company: "Acme" })).toBeNull();
  });

  it("returns null for invalid JSON string", () => {
    expect(parseEnrichmentData("not json")).toBeNull();
  });

  it("returns null for empty string", () => {
    expect(parseEnrichmentData("")).toBeNull();
  });

  it("returns null when enrichment is not an object", () => {
    expect(parseEnrichmentData({ enrichment: "string" })).toBeNull();
  });

  it("returns null for JSON array", () => {
    expect(parseEnrichmentData("[1]")).toBeNull();
  });

  it("returns null for JSON number string", () => {
    expect(parseEnrichmentData("42")).toBeNull();
  });

  it("returns null when enrichment is array", () => {
    expect(parseEnrichmentData({ enrichment: [1, 2] })).toBeNull();
  });

  it("returns null when enrichment is null", () => {
    expect(parseEnrichmentData({ enrichment: null })).toBeNull();
  });
});

describe("parseScoreBreakdown", () => {
  it("parses valid breakdown", () => {
    const result = parseScoreBreakdown({
      total: 80,
      rules: [{ name: "r1", description: "Rule 1", points: 30, matched: true }],
    });
    expect(result?.total).toBe(80);
    expect(result?.rules).toHaveLength(1);
    expect(result?.rules[0]?.name).toBe("r1");
  });

  it("returns null for non-object", () => {
    expect(parseScoreBreakdown("string")).toBeNull();
    expect(parseScoreBreakdown(null)).toBeNull();
    expect(parseScoreBreakdown(42)).toBeNull();
    expect(parseScoreBreakdown([1, 2])).toBeNull();
  });

  it("returns null when total is missing", () => {
    expect(parseScoreBreakdown({ rules: [] })).toBeNull();
  });

  it("returns null when rules is not array", () => {
    expect(parseScoreBreakdown({ total: 80, rules: "not-array" })).toBeNull();
  });

  it("skips invalid rules", () => {
    const result = parseScoreBreakdown({
      total: 50,
      rules: [
        { name: "r1", description: "d1", points: 10, matched: true },
        { name: 123 }, // invalid
        "not-object",
        null,
        [1, 2],
      ],
    });
    expect(result?.rules).toHaveLength(1);
  });
});

describe("filterLeadsByAssignee", () => {
  const cards = threadsToLeadCards([
    makeThread({ id: "t-1", metadata: '{"assigned_to":"alice"}' }),
    makeThread({ id: "t-2", metadata: '{"assigned_to":"bob"}' }),
    makeThread({ id: "t-3", metadata: "{}" }),
  ]);

  it("returns all for empty string", () => {
    expect(filterLeadsByAssignee(cards, "")).toHaveLength(3);
  });

  it("returns all for 'all'", () => {
    expect(filterLeadsByAssignee(cards, "all")).toHaveLength(3);
  });

  it("filters by assignee", () => {
    expect(filterLeadsByAssignee(cards, "alice")).toHaveLength(1);
  });

  it("returns empty for non-matching assignee", () => {
    expect(filterLeadsByAssignee(cards, "charlie")).toHaveLength(0);
  });
});

describe("filterLeadsByMinScore", () => {
  const cards = threadsToLeadCards([
    makeThread({ id: "t-1", metadata: '{"score":80}' }),
    makeThread({ id: "t-2", metadata: '{"score":40}' }),
    makeThread({ id: "t-3", metadata: "{}" }),
  ]);

  it("returns all for zero min score", () => {
    expect(filterLeadsByMinScore(cards, 0)).toHaveLength(3);
  });

  it("returns all for negative min score", () => {
    expect(filterLeadsByMinScore(cards, -10)).toHaveLength(3);
  });

  it("filters by min score", () => {
    expect(filterLeadsByMinScore(cards, 50)).toHaveLength(1);
  });
});

describe("getUniqueAssignees", () => {
  it("returns sorted unique assignees", () => {
    const cards = threadsToLeadCards([
      makeThread({ id: "t-1", metadata: '{"assigned_to":"charlie"}' }),
      makeThread({ id: "t-2", metadata: '{"assigned_to":"alice"}' }),
      makeThread({ id: "t-3", metadata: '{"assigned_to":"alice"}' }),
    ]);
    expect(getUniqueAssignees(cards)).toEqual(["alice", "charlie"]);
  });

  it("returns empty for no assignees", () => {
    const cards = threadsToLeadCards([makeThread({ metadata: "{}" })]);
    expect(getUniqueAssignees(cards)).toEqual([]);
  });
});

describe("isCrmActivityMessage", () => {
  const makeMsg = (type: string): Message => ({
    id: "m-1",
    thread_id: "t-1",
    body: "test",
    author_id: "u-1",
    metadata: "{}",
    type: type as Message["type"],
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  });

  it("returns true for CRM message types", () => {
    expect(isCrmActivityMessage(makeMsg("note"))).toBe(true);
    expect(isCrmActivityMessage(makeMsg("email"))).toBe(true);
    expect(isCrmActivityMessage(makeMsg("call_log"))).toBe(true);
    expect(isCrmActivityMessage(makeMsg("system"))).toBe(true);
  });

  it("returns false for non-CRM message types", () => {
    expect(isCrmActivityMessage(makeMsg("comment"))).toBe(false);
  });
});

describe("STAGE_PROBABILITIES", () => {
  it("has probabilities for all stages", () => {
    for (const stage of PIPELINE_STAGES) {
      expect(STAGE_PROBABILITIES[stage]).toBeDefined();
    }
  });

  it("has correct values for key stages", () => {
    expect(STAGE_PROBABILITIES.new_lead).toBe(5);
    expect(STAGE_PROBABILITIES.closed_won).toBe(100);
    expect(STAGE_PROBABILITIES.closed_lost).toBe(0);
    expect(STAGE_PROBABILITIES.negotiation).toBe(75);
  });
});

describe("calculateWeightedForecast", () => {
  it("calculates correctly", () => {
    expect(calculateWeightedForecast(100000, 50)).toBe(50000);
  });

  it("returns 0 for 0%", () => {
    expect(calculateWeightedForecast(100000, 0)).toBe(0);
  });

  it("returns full amount for 100%", () => {
    expect(calculateWeightedForecast(50000, 100)).toBe(50000);
  });

  it("rounds to integer", () => {
    expect(calculateWeightedForecast(33333, 33)).toBe(11000);
  });

  it("handles zero amount", () => {
    expect(calculateWeightedForecast(0, 50)).toBe(0);
  });
});

describe("getEffectiveProbability", () => {
  it("returns override when set", () => {
    expect(getEffectiveProbability("new_lead", 80)).toBe(80);
  });

  it("returns stage default when override is undefined", () => {
    expect(getEffectiveProbability("new_lead")).toBe(5);
    expect(getEffectiveProbability("negotiation")).toBe(75);
  });

  it("returns stage default when override is negative", () => {
    expect(getEffectiveProbability("proposal", -1)).toBe(50);
  });

  it("returns stage default when override is above 100", () => {
    expect(getEffectiveProbability("proposal", 101)).toBe(50);
  });

  it("returns 0 for override of 0", () => {
    expect(getEffectiveProbability("new_lead", 0)).toBe(0);
  });

  it("returns 100 for override of 100", () => {
    expect(getEffectiveProbability("new_lead", 100)).toBe(100);
  });
});

describe("isOverdue", () => {
  it("returns true for past dates", () => {
    expect(isOverdue("2020-01-01")).toBe(true);
  });

  it("returns false for future dates", () => {
    expect(isOverdue("2099-12-31")).toBe(false);
  });

  it("returns false for undefined", () => {
    expect(isOverdue(undefined)).toBe(false);
  });

  it("returns false for empty string", () => {
    expect(isOverdue("")).toBe(false);
  });
});

describe("daysBetween", () => {
  it("calculates positive days", () => {
    expect(daysBetween("2025-01-01", "2025-01-10")).toBe(9);
  });

  it("returns 0 for same day", () => {
    expect(daysBetween("2025-01-01", "2025-01-01")).toBe(0);
  });

  it("returns negative for future start", () => {
    expect(daysBetween("2025-01-10", "2025-01-01")).toBe(-9);
  });
});

describe("parseLeadData — extended fields", () => {
  it("parses deal_amount and weighted_forecast", () => {
    const data = parseLeadData({ deal_amount: 50000, weighted_forecast: 25000 });
    expect(data.deal_amount).toBe(50000);
    expect(data.weighted_forecast).toBe(25000);
  });

  it("parses expected_close_date and probability_override", () => {
    const data = parseLeadData({ expected_close_date: "2025-06-01", probability_override: 75 });
    expect(data.expected_close_date).toBe("2025-06-01");
    expect(data.probability_override).toBe(75);
  });

  it("parses opportunity_type and lead_source", () => {
    const data = parseLeadData({ opportunity_type: "new_business", lead_source: "website" });
    expect(data.opportunity_type).toBe("new_business");
    expect(data.lead_source).toBe("website");
  });

  it("parses crm_type", () => {
    const data = parseLeadData({ crm_type: "company" });
    expect(data.crm_type).toBe("company");
  });
});

describe("Enum constants", () => {
  it("OPPORTUNITY_TYPES has labels for all types", () => {
    for (const t of OPPORTUNITY_TYPES) {
      expect(OPPORTUNITY_TYPE_LABELS[t]).toBeDefined();
    }
  });

  it("LEAD_SOURCES has labels for all sources", () => {
    for (const s of LEAD_SOURCES) {
      expect(LEAD_SOURCE_LABELS[s]).toBeDefined();
    }
  });

  it("COMPANY_STATUSES has labels for all statuses", () => {
    for (const s of COMPANY_STATUSES) {
      expect(COMPANY_STATUS_LABELS[s]).toBeDefined();
    }
  });
});
