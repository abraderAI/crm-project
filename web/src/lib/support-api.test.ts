import { beforeEach, describe, expect, it, vi } from "vitest";

import { fetchDeftMembers, setTicketNotificationPref } from "./support-api";

const mockBuildUrl = vi.fn();
const mockBuildHeaders = vi.fn();
const mockParseResponse = vi.fn();
const mockFetch = vi.fn();

vi.mock("./api-client", () => ({
  buildUrl: (...args: unknown[]) => mockBuildUrl(...args),
  buildHeaders: (...args: unknown[]) => mockBuildHeaders(...args),
  parseResponse: (...args: unknown[]) => mockParseResponse(...args),
  clientMutate: vi.fn(),
}));

beforeEach(() => {
  vi.clearAllMocks();
  vi.stubGlobal("fetch", mockFetch);
  mockBuildUrl.mockReturnValue("http://localhost/v1/support/tickets/ticket-1/notifications");
  mockBuildHeaders.mockReturnValue({ Authorization: "Bearer token" });
});

describe("setTicketNotificationPref", () => {
  it("does not parse error payload for a 204 response", async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 204,
    } as Response);

    await setTicketNotificationPref("token", "ticket-1", "privacy");

    expect(mockParseResponse).not.toHaveBeenCalled();
  });

  it("parses error payload when response is not ok and not 204", async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 400,
    } as Response);

    await setTicketNotificationPref("token", "ticket-1", "full");

    expect(mockParseResponse).toHaveBeenCalledTimes(1);
  });
});

describe("fetchDeftMembers", () => {
  beforeEach(() => {
    mockBuildUrl.mockReturnValue("http://localhost/v1/support/deft-members");
  });

  it("calls fetch with correct URL and headers", async () => {
    const members = [{ user_id: "u1", display_name: "Alice", email: "a@deft.co" }];
    mockFetch.mockResolvedValue({ ok: true } as Response);
    mockParseResponse.mockResolvedValue({ data: members });

    const result = await fetchDeftMembers("my-token");

    expect(mockBuildUrl).toHaveBeenCalledWith("/support/deft-members");
    expect(mockBuildHeaders).toHaveBeenCalledWith("my-token");
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost/v1/support/deft-members",
      expect.objectContaining({ method: "GET", cache: "no-store" }),
    );
    expect(result).toEqual(members);
  });

  it("returns empty array when no members", async () => {
    mockFetch.mockResolvedValue({ ok: true } as Response);
    mockParseResponse.mockResolvedValue({ data: [] });

    const result = await fetchDeftMembers("token");
    expect(result).toEqual([]);
  });
});
