import { beforeEach, describe, expect, it, vi } from "vitest";

import { setTicketNotificationPref } from "./support-api";

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
