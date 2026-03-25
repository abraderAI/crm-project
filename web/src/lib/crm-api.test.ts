import { describe, expect, it, vi, beforeEach } from "vitest";
import {
  fetchCompanies,
  fetchCompany,
  createCompany,
  updateCompany,
  checkDuplicateCompany,
  fetchContacts,
  fetchContact,
  createContact,
  updateContact,
  checkDuplicateContact,
  fetchOpportunities,
  fetchOpportunity,
  createOpportunity,
  updateOpportunity,
  transitionOpportunity,
  reassignEntity,
  fetchEntityMessages,
  fetchLinkedContacts,
  fetchLinkedOpportunities,
} from "./crm-api";

const mockResponse = (data: unknown, ok = true, status = 200): Response =>
  ({
    ok,
    status,
    json: async () => data,
    statusText: ok ? "OK" : "Error",
  }) as unknown as Response;

describe("CRM API", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  // Company API
  describe("fetchCompanies", () => {
    it("fetches companies with correct URL", async () => {
      const spy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
        mockResponse({ data: [], page_info: { has_more: false } }),
      );
      const result = await fetchCompanies("token", "org1");
      expect(spy).toHaveBeenCalledOnce();
      expect(spy.mock.calls[0]![0]).toContain("/v1/orgs/org1/crm/companies");
      expect(result.data).toEqual([]);
    });

    it("passes query params", async () => {
      const spy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
        mockResponse({ data: [], page_info: { has_more: false } }),
      );
      await fetchCompanies("token", "org1", { limit: "10", cursor: "abc" });
      const url = spy.mock.calls[0]![0] as string;
      expect(url).toContain("limit=10");
      expect(url).toContain("cursor=abc");
    });
  });

  describe("fetchCompany", () => {
    it("fetches single company", async () => {
      const company = { id: "c1", title: "Acme" };
      vi.spyOn(globalThis, "fetch").mockResolvedValue(mockResponse(company));
      const result = await fetchCompany("token", "org1", "c1");
      expect(result).toEqual(company);
    });
  });

  describe("createCompany", () => {
    it("posts company data", async () => {
      const spy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
        mockResponse({ id: "c1", title: "New Co" }),
      );
      const result = await createCompany("token", "org1", { title: "New Co" });
      expect(result.id).toBe("c1");
      const opts = spy.mock.calls[0]![1] as RequestInit;
      expect(opts.method).toBe("POST");
    });
  });

  describe("updateCompany", () => {
    it("patches company data", async () => {
      const spy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
        mockResponse({ id: "c1", title: "Updated Co" }),
      );
      await updateCompany("token", "org1", "c1", { title: "Updated Co" });
      const opts = spy.mock.calls[0]![1] as RequestInit;
      expect(opts.method).toBe("PATCH");
    });
  });

  describe("checkDuplicateCompany", () => {
    it("returns matching companies", async () => {
      vi.spyOn(globalThis, "fetch").mockResolvedValue(
        mockResponse({ data: [{ id: "c1", title: "Acme" }] }),
      );
      const result = await checkDuplicateCompany("token", "org1", "Acme");
      expect(result).toHaveLength(1);
      expect(result[0]!.title).toBe("Acme");
    });
  });

  // Contact API
  describe("fetchContacts", () => {
    it("fetches contacts", async () => {
      vi.spyOn(globalThis, "fetch").mockResolvedValue(
        mockResponse({ data: [], page_info: { has_more: false } }),
      );
      const result = await fetchContacts("token", "org1");
      expect(result.data).toEqual([]);
    });
  });

  describe("createContact", () => {
    it("posts contact data", async () => {
      const spy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
        mockResponse({ id: "ct1", title: "John" }),
      );
      const result = await createContact("token", "org1", { title: "John" });
      expect(result.id).toBe("ct1");
      expect((spy.mock.calls[0]![1] as RequestInit).method).toBe("POST");
    });
  });

  describe("checkDuplicateContact", () => {
    it("returns matching contacts by email", async () => {
      vi.spyOn(globalThis, "fetch").mockResolvedValue(
        mockResponse({ data: [{ id: "ct1", title: "John" }] }),
      );
      const result = await checkDuplicateContact("token", "org1", "john@test.com");
      expect(result).toHaveLength(1);
    });
  });

  describe("fetchContact", () => {
    it("fetches single contact", async () => {
      const contact = { id: "ct1", title: "Jane" };
      vi.spyOn(globalThis, "fetch").mockResolvedValue(mockResponse(contact));
      const result = await fetchContact("token", "org1", "ct1");
      expect(result).toEqual(contact);
    });
  });

  describe("updateContact", () => {
    it("patches contact data", async () => {
      const spy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
        mockResponse({ id: "ct1", title: "Updated" }),
      );
      await updateContact("token", "org1", "ct1", { title: "Updated" });
      const opts = spy.mock.calls[0]![1] as RequestInit;
      expect(opts.method).toBe("PATCH");
    });
  });

  // Opportunity API
  describe("fetchOpportunities", () => {
    it("fetches opportunities", async () => {
      vi.spyOn(globalThis, "fetch").mockResolvedValue(
        mockResponse({ data: [], page_info: { has_more: false } }),
      );
      const result = await fetchOpportunities("token", "org1");
      expect(result.data).toEqual([]);
    });
  });

  describe("fetchOpportunity", () => {
    it("fetches single opportunity", async () => {
      const opp = { id: "o1", title: "Deal" };
      vi.spyOn(globalThis, "fetch").mockResolvedValue(mockResponse(opp));
      const result = await fetchOpportunity("token", "org1", "o1");
      expect(result).toEqual(opp);
    });
  });

  describe("createOpportunity", () => {
    it("posts opportunity data", async () => {
      const spy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
        mockResponse({ id: "o1", title: "New Opp" }),
      );
      const result = await createOpportunity("token", "org1", { title: "New Opp" });
      expect(result.id).toBe("o1");
      expect((spy.mock.calls[0]![1] as RequestInit).method).toBe("POST");
    });
  });

  describe("updateOpportunity", () => {
    it("patches opportunity data", async () => {
      const spy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
        mockResponse({ id: "o1", title: "Updated" }),
      );
      await updateOpportunity("token", "org1", "o1", { title: "Updated" });
      expect((spy.mock.calls[0]![1] as RequestInit).method).toBe("PATCH");
    });
  });

  describe("transitionOpportunity", () => {
    it("posts transition with stage and reason", async () => {
      const spy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
        mockResponse({ id: "o1", stage: "qualified" }),
      );
      await transitionOpportunity("token", "org1", "o1", {
        stage: "qualified",
        reason: "test",
      });
      const opts = spy.mock.calls[0]![1] as RequestInit;
      expect(opts.method).toBe("POST");
      expect(opts.body).toContain("qualified");
    });
  });

  describe("reassignEntity", () => {
    it("posts reassignment", async () => {
      const spy = vi.spyOn(globalThis, "fetch").mockResolvedValue(mockResponse(null));
      await reassignEntity("token", "org1", "companies", "c1", "user2");
      const opts = spy.mock.calls[0]![1] as RequestInit;
      expect(opts.method).toBe("POST");
      expect(opts.body).toContain("user2");
    });
  });

  describe("fetchEntityMessages", () => {
    it("fetches messages for entity", async () => {
      vi.spyOn(globalThis, "fetch").mockResolvedValue(
        mockResponse({ data: [], page_info: { has_more: false } }),
      );
      const result = await fetchEntityMessages("token", "org1", "companies", "c1");
      expect(result.data).toEqual([]);
    });
  });

  describe("fetchLinkedContacts", () => {
    it("fetches linked contacts for company", async () => {
      vi.spyOn(globalThis, "fetch").mockResolvedValue(
        mockResponse({ data: [], page_info: { has_more: false } }),
      );
      const result = await fetchLinkedContacts("token", "org1", "c1");
      expect(result.data).toEqual([]);
    });
  });

  describe("fetchLinkedOpportunities", () => {
    it("fetches linked opps", async () => {
      vi.spyOn(globalThis, "fetch").mockResolvedValue(
        mockResponse({ data: [], page_info: { has_more: false } }),
      );
      const result = await fetchLinkedOpportunities("token", "org1", "contacts", "ct1");
      expect(result.data).toEqual([]);
    });
  });
});
