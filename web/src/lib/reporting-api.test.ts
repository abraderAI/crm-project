import { describe, expect, it, vi, beforeEach } from "vitest";

// Mock @clerk/nextjs/server before imports.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs/server", () => ({
  auth: vi.fn().mockResolvedValue({ getToken: () => mockGetToken() }),
}));

// Mock api-client functions.
const mockBuildUrl = vi.fn();
const mockBuildHeaders = vi.fn();
const mockParseResponse = vi.fn();
vi.mock("./api-client", () => ({
  buildUrl: (...args: unknown[]) => mockBuildUrl(...args),
  buildHeaders: (...args: unknown[]) => mockBuildHeaders(...args),
  parseResponse: (...args: unknown[]) => mockParseResponse(...args),
}));

// Mock global fetch.
const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

import {
  getSupportMetrics,
  getSalesMetrics,
  getAdminSupportMetrics,
  getAdminSalesMetrics,
  getSupportExportUrl,
  getSalesExportUrl,
  getAdminSupportExportUrl,
  getAdminSalesExportUrl,
} from "./reporting-api";

describe("reporting-api", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
    mockBuildUrl.mockReturnValue("http://localhost:8080/v1/test");
    mockBuildHeaders.mockReturnValue({ Authorization: "Bearer test-token" });
    mockFetch.mockResolvedValue(new Response());
  });

  describe("getSupportMetrics", () => {
    it("fetches org-scoped support metrics with params", async () => {
      const metrics = { overdue_count: 3, avg_resolution_hours: 12 };
      mockParseResponse.mockResolvedValue(metrics);

      const result = await getSupportMetrics("my-org", { from: "2026-01-01", to: "2026-01-31" });

      expect(mockBuildUrl).toHaveBeenCalledWith("/orgs/my-org/reports/support", {
        from: "2026-01-01",
        to: "2026-01-31",
      });
      expect(mockBuildHeaders).toHaveBeenCalledWith("test-token");
      expect(mockFetch).toHaveBeenCalled();
      expect(result).toEqual(metrics);
    });

    it("throws when unauthenticated", async () => {
      mockGetToken.mockResolvedValue(null);
      await expect(getSupportMetrics("org1", {})).rejects.toThrow("Unauthenticated");
    });
  });

  describe("getSalesMetrics", () => {
    it("fetches org-scoped sales metrics with params", async () => {
      const metrics = { win_rate: 0.65, loss_rate: 0.15 };
      mockParseResponse.mockResolvedValue(metrics);

      const result = await getSalesMetrics("my-org", { assignee: "user-1" });

      expect(mockBuildUrl).toHaveBeenCalledWith("/orgs/my-org/reports/sales", {
        assignee: "user-1",
      });
      expect(result).toEqual(metrics);
    });

    it("passes empty params record for empty ReportParams", async () => {
      mockParseResponse.mockResolvedValue({});
      await getSalesMetrics("org1", {});
      expect(mockBuildUrl).toHaveBeenCalledWith("/orgs/org1/reports/sales", {});
    });
  });

  describe("getAdminSupportMetrics", () => {
    it("fetches admin support metrics", async () => {
      const metrics = { org_breakdown: [], overdue_count: 5 };
      mockParseResponse.mockResolvedValue(metrics);

      const result = await getAdminSupportMetrics({ from: "2026-03-01" });

      expect(mockBuildUrl).toHaveBeenCalledWith("/admin/reports/support", {
        from: "2026-03-01",
      });
      expect(result).toEqual(metrics);
    });
  });

  describe("getAdminSalesMetrics", () => {
    it("fetches admin sales metrics", async () => {
      const metrics = { org_breakdown: [], win_rate: 0.5 };
      mockParseResponse.mockResolvedValue(metrics);

      const result = await getAdminSalesMetrics({ to: "2026-03-31" });

      expect(mockBuildUrl).toHaveBeenCalledWith("/admin/reports/sales", {
        to: "2026-03-31",
      });
      expect(result).toEqual(metrics);
    });
  });

  describe("getSupportExportUrl", () => {
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
