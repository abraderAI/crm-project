import { describe, expect, it } from "vitest";
import { getDefaultLayout, validateLayout, WIDGET_IDS, VALID_WIDGET_IDS } from "./default-layouts";
import type { Tier } from "./tier-types";

describe("default-layouts", () => {
  describe("WIDGET_IDS", () => {
    it("has all expected widget IDs defined", () => {
      expect(WIDGET_IDS.DOCS_HIGHLIGHTS).toBe("docs-highlights");
      expect(WIDGET_IDS.SYSTEM_HEALTH).toBe("system-health");
      expect(WIDGET_IDS.LEAD_PIPELINE).toBe("lead-pipeline");
    });
  });

  describe("VALID_WIDGET_IDS", () => {
    it("contains all WIDGET_IDS values", () => {
      for (const id of Object.values(WIDGET_IDS)) {
        expect(VALID_WIDGET_IDS.has(id)).toBe(true);
      }
    });

    it("does not contain unknown IDs", () => {
      expect(VALID_WIDGET_IDS.has("nonexistent")).toBe(false);
    });
  });

  describe("getDefaultLayout", () => {
    it("returns Tier 1 default layout", () => {
      const layout = getDefaultLayout(1);
      expect(layout).toHaveLength(3);
      expect(layout[0]?.widget_id).toBe(WIDGET_IDS.DOCS_HIGHLIGHTS);
      expect(layout[1]?.widget_id).toBe(WIDGET_IDS.FORUM_HIGHLIGHTS);
      expect(layout[2]?.widget_id).toBe(WIDGET_IDS.GET_STARTED);
      expect(layout.every((w) => w.visible)).toBe(true);
    });

    it("returns Tier 2 default layout", () => {
      const layout = getDefaultLayout(2);
      expect(layout).toHaveLength(4);
      expect(layout[0]?.widget_id).toBe(WIDGET_IDS.MY_PROFILE);
      expect(layout[3]?.widget_id).toBe(WIDGET_IDS.UPGRADE_CTA);
    });

    it("returns Tier 3 default layout", () => {
      const layout = getDefaultLayout(3);
      expect(layout).toHaveLength(3);
      expect(layout[0]?.widget_id).toBe(WIDGET_IDS.ORG_OVERVIEW);
    });

    it("returns Tier 4 sales default when no department specified", () => {
      const layout = getDefaultLayout(4);
      expect(layout[0]?.widget_id).toBe(WIDGET_IDS.LEAD_PIPELINE);
    });

    it("returns Tier 4 sales layout for sales department", () => {
      const layout = getDefaultLayout(4, "sales");
      expect(layout).toHaveLength(3);
      expect(layout[0]?.widget_id).toBe(WIDGET_IDS.LEAD_PIPELINE);
    });

    it("returns Tier 4 support layout for support department", () => {
      const layout = getDefaultLayout(4, "support");
      expect(layout).toHaveLength(2);
      expect(layout[0]?.widget_id).toBe(WIDGET_IDS.TICKET_QUEUE);
    });

    it("returns Tier 4 finance layout for finance department", () => {
      const layout = getDefaultLayout(4, "finance");
      expect(layout).toHaveLength(1);
      expect(layout[0]?.widget_id).toBe(WIDGET_IDS.BILLING_OVERVIEW);
    });

    it("returns Tier 5 default layout", () => {
      const layout = getDefaultLayout(5);
      expect(layout).toHaveLength(4);
      expect(layout[0]?.widget_id).toBe(WIDGET_IDS.ORG_ACCESS_CONTROL);
    });

    it("returns empty Tier 6 default layout (admin console has real data)", () => {
      const layout = getDefaultLayout(6);
      expect(layout).toHaveLength(0);
    });

    it("returns a copy, not a reference to the internal array", () => {
      const layout1 = getDefaultLayout(1);
      const layout2 = getDefaultLayout(1);
      expect(layout1).toEqual(layout2);
      expect(layout1).not.toBe(layout2);
    });

    it("falls back to Tier 1 default for unknown tier", () => {
      const layout = getDefaultLayout(99 as Tier);
      expect(layout).toEqual(getDefaultLayout(1));
    });

    it("ignores null department for Tier 4", () => {
      const layout = getDefaultLayout(4, null);
      expect(layout[0]?.widget_id).toBe(WIDGET_IDS.LEAD_PIPELINE);
    });
  });

  describe("validateLayout", () => {
    it("returns empty array for valid layout", () => {
      const layout = getDefaultLayout(1);
      expect(validateLayout(layout)).toEqual([]);
    });

    it("reports unknown widget IDs", () => {
      const errors = validateLayout([{ widget_id: "unknown-widget", visible: true }]);
      expect(errors).toHaveLength(1);
      expect(errors[0]).toContain("Unknown widget ID");
    });

    it("reports duplicate widget IDs", () => {
      const errors = validateLayout([
        { widget_id: WIDGET_IDS.DOCS_HIGHLIGHTS, visible: true },
        { widget_id: WIDGET_IDS.DOCS_HIGHLIGHTS, visible: false },
      ]);
      expect(errors).toHaveLength(1);
      expect(errors[0]).toContain("Duplicate widget ID");
    });

    it("reports both unknown and duplicate errors", () => {
      const errors = validateLayout([
        { widget_id: "bad", visible: true },
        { widget_id: "bad", visible: true },
      ]);
      // 2 unknown + 1 duplicate = 3 errors.
      expect(errors).toHaveLength(3);
    });

    it("accepts empty layout", () => {
      expect(validateLayout([])).toEqual([]);
    });
  });
});
