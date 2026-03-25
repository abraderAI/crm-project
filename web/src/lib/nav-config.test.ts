import { describe, expect, it } from "vitest";
import { getNavItemsForTier, SIDEBAR_NAV_ITEMS } from "./nav-config";
import type { Tier } from "./tier-types";

/** Helper: extract top-level item IDs from filtered nav. */
function ids(tier: Tier): string[] {
  return getNavItemsForTier(tier).map((i) => i.id);
}

/** Helper: check if a specific item exists in filtered nav. */
function hasItem(tier: Tier, id: string): boolean {
  return getNavItemsForTier(tier).some((i) => i.id === id);
}

/** Helper: get child IDs for a parent item at a given tier. */
function childIds(tier: Tier, parentId: string): string[] {
  const parent = getNavItemsForTier(tier).find((i) => i.id === parentId);
  return parent?.children?.map((c) => c.id) ?? [];
}

describe("SIDEBAR_NAV_ITEMS", () => {
  it("has a non-empty list of items", () => {
    expect(SIDEBAR_NAV_ITEMS.length).toBeGreaterThan(0);
  });

  it("every item has required fields", () => {
    for (const item of SIDEBAR_NAV_ITEMS) {
      expect(item.id).toBeTruthy();
      expect(item.label).toBeTruthy();
      expect(item.href).toBeTruthy();
      expect(item.icon).toBeDefined();
      expect(item.minTier).toBeGreaterThanOrEqual(1);
      expect(item.minTier).toBeLessThanOrEqual(6);
    }
  });

  it("has unique IDs across all items including children", () => {
    const allIds: string[] = [];
    for (const item of SIDEBAR_NAV_ITEMS) {
      allIds.push(item.id);
      if (item.children) {
        for (const child of item.children) {
          allIds.push(child.id);
        }
      }
    }
    expect(new Set(allIds).size).toBe(allIds.length);
  });
});

describe("getNavItemsForTier", () => {
  describe("Tier 1 (Anonymous)", () => {
    it("sees only Home, Forum, Docs", () => {
      expect(ids(1)).toEqual(["home", "forum", "docs"]);
    });

    it("does not see Support", () => {
      expect(hasItem(1, "support")).toBe(false);
    });

    it("does not see Admin", () => {
      expect(hasItem(1, "admin")).toBe(false);
    });
  });

  describe("Tier 2 (Registered Developer)", () => {
    it("sees tier 1 items plus Support, Notifications, Search", () => {
      const items = ids(2);
      expect(items).toContain("home");
      expect(items).toContain("forum");
      expect(items).toContain("docs");
      expect(items).toContain("support");
      expect(items).toContain("notifications");
      expect(items).toContain("search");
    });

    it("does not see Settings, CRM, Reports, Admin", () => {
      expect(hasItem(2, "settings")).toBe(false);
      expect(hasItem(2, "crm")).toBe(false);
      expect(hasItem(2, "reports")).toBe(false);
      expect(hasItem(2, "admin")).toBe(false);
    });

    it("has Support sub-menu with All Tickets and New Ticket", () => {
      expect(childIds(2, "support")).toEqual(["support-tickets", "support-new"]);
    });
  });

  describe("Tier 3 (Paying Customer)", () => {
    it("includes Settings", () => {
      expect(hasItem(3, "settings")).toBe(true);
    });

    it("does not see CRM or Admin", () => {
      expect(hasItem(3, "crm")).toBe(false);
      expect(hasItem(3, "admin")).toBe(false);
    });
  });

  describe("Tier 4 (DEFT Employee)", () => {
    it("sees CRM and Reports", () => {
      expect(hasItem(4, "crm")).toBe(true);
      expect(hasItem(4, "reports")).toBe(true);
    });

    it("has CRM sub-menu with Companies, Contacts, Pipeline, Inbox, Import, Leads", () => {
      expect(childIds(4, "crm")).toEqual([
        "crm-companies",
        "crm-contacts",
        "crm-pipeline",
        "crm-inbox",
        "crm-import",
        "crm-leads",
      ]);
    });

    it("has Reports sub-menu", () => {
      expect(childIds(4, "reports")).toEqual(["reports-support", "reports-sales"]);
    });

    it("does not see Admin", () => {
      expect(hasItem(4, "admin")).toBe(false);
    });
  });

  describe("Tier 5 (Customer Org Admin)", () => {
    it("sees same items as tier 4 (no Admin)", () => {
      expect(hasItem(5, "admin")).toBe(false);
      expect(hasItem(5, "crm")).toBe(true);
    });
  });

  describe("Tier 6 (Platform Admin)", () => {
    it("sees Admin", () => {
      expect(hasItem(6, "admin")).toBe(true);
    });

    it("has all 19 admin sub-items", () => {
      const adminChildren = childIds(6, "admin");
      expect(adminChildren.length).toBe(19);
      expect(adminChildren).toContain("admin-overview");
      expect(adminChildren).toContain("admin-orgs");
      expect(adminChildren).toContain("admin-users");
      expect(adminChildren).toContain("admin-settings");
      expect(adminChildren).toContain("admin-webhooks");
      expect(adminChildren).toContain("admin-billing");
      expect(adminChildren).toContain("admin-forums");
    });

    it("sees all items from every tier", () => {
      const items = ids(6);
      expect(items).toContain("home");
      expect(items).toContain("support");
      expect(items).toContain("settings");
      expect(items).toContain("crm");
      expect(items).toContain("reports");
      expect(items).toContain("admin");
    });
  });

  it("higher tiers always see a superset of lower tier items", () => {
    const tiers: Tier[] = [1, 2, 3, 4, 5, 6];
    for (let i = 0; i < tiers.length - 1; i++) {
      const lowerIds = new Set(ids(tiers[i]));
      const higherIds = new Set(ids(tiers[i + 1]));
      for (const id of lowerIds) {
        expect(higherIds.has(id)).toBe(true);
      }
    }
  });

  it("does not mutate the original SIDEBAR_NAV_ITEMS", () => {
    const before = JSON.stringify(SIDEBAR_NAV_ITEMS);
    getNavItemsForTier(1);
    getNavItemsForTier(6);
    expect(JSON.stringify(SIDEBAR_NAV_ITEMS)).toBe(before);
  });

  it("omits children array when all children are filtered out", () => {
    // Tier 1 should not see admin, but even if we had a hypothetical parent
    // at tier 1 with children at tier 6, the children should be omitted.
    // Test that parents without qualifying children have children=undefined.
    const tier1Items = getNavItemsForTier(1);
    for (const item of tier1Items) {
      // At tier 1, no item should have children since all child items require tier 2+.
      expect(item.children).toBeUndefined();
    }
  });

  it("items without children have children=undefined at all tiers", () => {
    const tiers: Tier[] = [1, 2, 3, 4, 5, 6];
    for (const tier of tiers) {
      const items = getNavItemsForTier(tier);
      const home = items.find((i) => i.id === "home");
      expect(home?.children).toBeUndefined();
    }
  });
});
