import { describe, expect, it } from "vitest";

import {
  getSupportExportUrl,
  getSalesExportUrl,
  getAdminSupportExportUrl,
  getAdminSalesExportUrl,
} from "./reporting-api";

describe("reporting-api export URL builders", () => {
  describe("getSupportExportUrl",
    it("builds export URL with all params", () => {
      const url = getSupportExportUrl("org1", {
        from: "2026-01-01",
        to: "2026-01-31",
        assignee: "u1",
      });
      expect(url).toContain("/v1/orgs/org1/reports/support/export");
      expect(url).toContain("from=2026-01-01");
      expect(url).toContain("to=2026-01-31");
      expect(url).toContain("assignee=u1");
    });

    it("builds export URL with no params", () => {
      const url = getSupportExportUrl("org1", {});
      expect(url).toBe("http://localhost:8080/v1/orgs/org1/reports/support/export");
    });
  });

  describe("getSalesExportUrl", () => {
    it("builds export URL with from param", () => {
      const url = getSalesExportUrl("org1", { from: "2026-02-01" });
      expect(url).toContain("/v1/orgs/org1/reports/sales/export");
      expect(url).toContain("from=2026-02-01");
    });
  });

  describe("getAdminSupportExportUrl", () => {
    it("builds admin export URL", () => {
      const url = getAdminSupportExportUrl({ from: "2026-03-01", to: "2026-03-31" });
      expect(url).toContain("/v1/admin/reports/support/export");
      expect(url).toContain("from=2026-03-01");
      expect(url).toContain("to=2026-03-31");
    });

    it("builds admin export URL with no params", () => {
      const url = getAdminSupportExportUrl({});
      expect(url).toBe("http://localhost:8080/v1/admin/reports/support/export");
    });
  });

  describe("getAdminSalesExportUrl", () => {
    it("builds admin sales export URL", () => {
      const url = getAdminSalesExportUrl({ assignee: "u2" });
      expect(url).toContain("/v1/admin/reports/sales/export");
      expect(url).toContain("assignee=u2");
    });

    it("builds admin sales export URL with no params", () => {
      const url = getAdminSalesExportUrl({});
      expect(url).toBe("http://localhost:8080/v1/admin/reports/sales/export");
    });
  });
});
