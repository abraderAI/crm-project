import { render, screen, within, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach, type Mock } from "vitest";
import { PhoneNumberManager } from "./phone-number-manager";
import type { PhoneNumber, PhoneNumberSearchResult } from "@/lib/api-types";

/* ------------------------------------------------------------------ */
/*  Fixtures                                                          */
/* ------------------------------------------------------------------ */

const ownedNumbers: PhoneNumber[] = [
  {
    phone_number: "+15551234567",
    status: "active",
    dispatch_rule_id: "rule-1",
    purchased_at: "2026-01-15T10:00:00Z",
  },
  {
    phone_number: "+15559876543",
    status: "pending",
    dispatch_rule_id: "rule-2",
    purchased_at: "2026-02-20T14:30:00Z",
  },
];

const searchResults: PhoneNumberSearchResult[] = [
  {
    phone_number: "+15550001111",
    country: "US",
    area_code: "555",
    monthly_cost: "$1.00",
  },
  {
    phone_number: "+15550002222",
    country: "US",
    area_code: "555",
    monthly_cost: "$1.00",
  },
];

/* ------------------------------------------------------------------ */
/*  Mock fetch globally                                               */
/* ------------------------------------------------------------------ */

const mockFetch = vi.fn() as Mock;

beforeEach(() => {
  vi.resetAllMocks();
  global.fetch = mockFetch;
  // Default: owned numbers list returns empty
  mockFetch.mockResolvedValue({
    ok: true,
    json: async () => ({ data: [] }),
  });
});

/* ------------------------------------------------------------------ */
/*  Helpers                                                           */
/* ------------------------------------------------------------------ */

/** Set up fetch to return owned numbers on GET and optionally search results on POST. */
function setupFetch(opts?: {
  owned?: PhoneNumber[];
  search?: PhoneNumberSearchResult[];
  purchaseOk?: boolean;
}): void {
  mockFetch.mockImplementation(async (url: string, init?: RequestInit) => {
    const method = init?.method ?? "GET";
    if (typeof url === "string" && url.includes("/numbers/search") && method === "POST") {
      return {
        ok: true,
        json: async () => ({ data: opts?.search ?? [] }),
      };
    }
    if (typeof url === "string" && url.includes("/numbers/purchase") && method === "POST") {
      return {
        ok: opts?.purchaseOk !== false,
        json: async () =>
          opts?.purchaseOk !== false
            ? { phone_number: "+15550001111", status: "active" }
            : { title: "Purchase failed", status: 400 },
        status: opts?.purchaseOk !== false ? 200 : 400,
      };
    }
    // Default: GET owned numbers
    return {
      ok: true,
      json: async () => ({ data: opts?.owned ?? [] }),
    };
  });
}

/* ------------------------------------------------------------------ */
/*  Tests                                                             */
/* ------------------------------------------------------------------ */

describe("PhoneNumberManager", () => {
  it("renders the component with heading", () => {
    render(<PhoneNumberManager org="org1" />);
    expect(screen.getByTestId("phone-number-manager")).toBeInTheDocument();
    expect(screen.getByText("Phone Number Management")).toBeInTheDocument();
  });

  it("renders the owned numbers section heading", () => {
    render(<PhoneNumberManager org="org1" />);
    expect(screen.getByText("Owned Numbers")).toBeInTheDocument();
  });

  it("renders the search available section heading", () => {
    render(<PhoneNumberManager org="org1" />);
    expect(screen.getByText("Search Available Numbers")).toBeInTheDocument();
  });

  it("shows empty state when no owned numbers", async () => {
    setupFetch({ owned: [] });
    render(<PhoneNumberManager org="org1" />);
    expect(await screen.findByTestId("owned-numbers-empty")).toBeInTheDocument();
  });

  it("renders owned numbers table with data", async () => {
    setupFetch({ owned: ownedNumbers });
    render(<PhoneNumberManager org="org1" />);
    expect(await screen.findByText("+15551234567")).toBeInTheDocument();
    expect(screen.getByText("+15559876543")).toBeInTheDocument();
  });

  it("displays status for each owned number", async () => {
    setupFetch({ owned: ownedNumbers });
    render(<PhoneNumberManager org="org1" />);
    expect(await screen.findByText("active")).toBeInTheDocument();
    expect(screen.getByText("pending")).toBeInTheDocument();
  });

  it("displays dispatch rule for each owned number", async () => {
    setupFetch({ owned: ownedNumbers });
    render(<PhoneNumberManager org="org1" />);
    expect(await screen.findByText("rule-1")).toBeInTheDocument();
    expect(screen.getByText("rule-2")).toBeInTheDocument();
  });

  it("renders area code input", () => {
    render(<PhoneNumberManager org="org1" />);
    expect(screen.getByTestId("search-area-code")).toBeInTheDocument();
  });

  it("renders country selector", () => {
    render(<PhoneNumberManager org="org1" />);
    expect(screen.getByTestId("search-country")).toBeInTheDocument();
  });

  it("renders search button", () => {
    render(<PhoneNumberManager org="org1" />);
    expect(screen.getByTestId("search-btn")).toBeInTheDocument();
  });

  it("triggers search API call on search button click", async () => {
    const user = userEvent.setup();
    setupFetch({ search: searchResults });
    render(<PhoneNumberManager org="org1" />);

    await user.type(screen.getByTestId("search-area-code"), "555");
    await user.click(screen.getByTestId("search-btn"));

    await waitFor(() => {
      const calls = mockFetch.mock.calls;
      const searchCall = calls.find(
        (c: unknown[]) => typeof c[0] === "string" && c[0].includes("/numbers/search"),
      );
      expect(searchCall).toBeDefined();
    });
  });

  it("displays search results after search", async () => {
    const user = userEvent.setup();
    setupFetch({ search: searchResults });
    render(<PhoneNumberManager org="org1" />);

    await user.type(screen.getByTestId("search-area-code"), "555");
    await user.click(screen.getByTestId("search-btn"));

    expect(await screen.findByText("+15550001111")).toBeInTheDocument();
    expect(screen.getByText("+15550002222")).toBeInTheDocument();
  });

  it("renders purchase button for each search result", async () => {
    const user = userEvent.setup();
    setupFetch({ search: searchResults });
    render(<PhoneNumberManager org="org1" />);

    await user.type(screen.getByTestId("search-area-code"), "555");
    await user.click(screen.getByTestId("search-btn"));

    const buttons = await screen.findAllByTestId("purchase-btn");
    expect(buttons).toHaveLength(2);
  });

  it("opens confirmation modal when purchase button clicked", async () => {
    const user = userEvent.setup();
    setupFetch({ search: searchResults });
    render(<PhoneNumberManager org="org1" />);

    await user.type(screen.getByTestId("search-area-code"), "555");
    await user.click(screen.getByTestId("search-btn"));

    const buttons = await screen.findAllByTestId("purchase-btn");
    await user.click(buttons[0]!);

    expect(screen.getByTestId("purchase-confirm-modal")).toBeInTheDocument();
  });

  it("shows billable action warning in confirmation modal", async () => {
    const user = userEvent.setup();
    setupFetch({ search: searchResults });
    render(<PhoneNumberManager org="org1" />);

    await user.type(screen.getByTestId("search-area-code"), "555");
    await user.click(screen.getByTestId("search-btn"));

    const buttons = await screen.findAllByTestId("purchase-btn");
    await user.click(buttons[0]!);

    expect(screen.getByText(/may incur charges/i)).toBeInTheDocument();
  });

  it("shows the phone number being purchased in modal", async () => {
    const user = userEvent.setup();
    setupFetch({ search: searchResults });
    render(<PhoneNumberManager org="org1" />);

    await user.type(screen.getByTestId("search-area-code"), "555");
    await user.click(screen.getByTestId("search-btn"));

    const buttons = await screen.findAllByTestId("purchase-btn");
    await user.click(buttons[0]!);

    const modal = screen.getByTestId("purchase-confirm-modal");
    expect(within(modal).getByText("+15550001111")).toBeInTheDocument();
  });

  it("closes confirmation modal on cancel", async () => {
    const user = userEvent.setup();
    setupFetch({ search: searchResults });
    render(<PhoneNumberManager org="org1" />);

    await user.type(screen.getByTestId("search-area-code"), "555");
    await user.click(screen.getByTestId("search-btn"));

    const buttons = await screen.findAllByTestId("purchase-btn");
    await user.click(buttons[0]!);

    expect(screen.getByTestId("purchase-confirm-modal")).toBeInTheDocument();
    await user.click(screen.getByTestId("purchase-cancel-btn"));
    expect(screen.queryByTestId("purchase-confirm-modal")).not.toBeInTheDocument();
  });

  it("calls purchase API on confirm", async () => {
    const user = userEvent.setup();
    setupFetch({ search: searchResults, purchaseOk: true });
    render(<PhoneNumberManager org="org1" />);

    await user.type(screen.getByTestId("search-area-code"), "555");
    await user.click(screen.getByTestId("search-btn"));

    const buttons = await screen.findAllByTestId("purchase-btn");
    await user.click(buttons[0]!);
    await user.click(screen.getByTestId("purchase-confirm-btn"));

    await waitFor(() => {
      const calls = mockFetch.mock.calls;
      const purchaseCall = calls.find(
        (c: unknown[]) => typeof c[0] === "string" && c[0].includes("/numbers/purchase"),
      );
      expect(purchaseCall).toBeDefined();
    });
  });

  it("sends correct phone number in purchase request body", async () => {
    const user = userEvent.setup();
    setupFetch({ search: searchResults, purchaseOk: true });
    render(<PhoneNumberManager org="org1" />);

    await user.type(screen.getByTestId("search-area-code"), "555");
    await user.click(screen.getByTestId("search-btn"));

    const buttons = await screen.findAllByTestId("purchase-btn");
    await user.click(buttons[0]!);
    await user.click(screen.getByTestId("purchase-confirm-btn"));

    await waitFor(() => {
      const calls = mockFetch.mock.calls;
      const purchaseCall = calls.find(
        (c: unknown[]) => typeof c[0] === "string" && c[0].includes("/numbers/purchase"),
      );
      expect(purchaseCall).toBeDefined();
      const body = JSON.parse(purchaseCall![1]?.body as string) as { phone_number: string };
      expect(body.phone_number).toBe("+15550001111");
    });
  });

  it("closes modal after successful purchase", async () => {
    const user = userEvent.setup();
    setupFetch({ search: searchResults, purchaseOk: true });
    render(<PhoneNumberManager org="org1" />);

    await user.type(screen.getByTestId("search-area-code"), "555");
    await user.click(screen.getByTestId("search-btn"));

    const buttons = await screen.findAllByTestId("purchase-btn");
    await user.click(buttons[0]!);
    await user.click(screen.getByTestId("purchase-confirm-btn"));

    await waitFor(() => {
      expect(screen.queryByTestId("purchase-confirm-modal")).not.toBeInTheDocument();
    });
  });

  it("shows error on purchase failure", async () => {
    const user = userEvent.setup();
    setupFetch({ search: searchResults, purchaseOk: false });
    render(<PhoneNumberManager org="org1" />);

    await user.type(screen.getByTestId("search-area-code"), "555");
    await user.click(screen.getByTestId("search-btn"));

    const buttons = await screen.findAllByTestId("purchase-btn");
    await user.click(buttons[0]!);
    await user.click(screen.getByTestId("purchase-confirm-btn"));

    expect(await screen.findByTestId("purchase-error")).toBeInTheDocument();
  });

  it("shows searching state while search is in progress", async () => {
    const user = userEvent.setup();
    let resolveSearch: ((value: unknown) => void) | undefined;
    mockFetch.mockImplementation(async (url: string, init?: RequestInit) => {
      const method = init?.method ?? "GET";
      if (typeof url === "string" && url.includes("/numbers/search") && method === "POST") {
        return new Promise((resolve) => {
          resolveSearch = resolve;
        });
      }
      return { ok: true, json: async () => ({ data: [] }) };
    });

    render(<PhoneNumberManager org="org1" />);
    await user.type(screen.getByTestId("search-area-code"), "555");
    await user.click(screen.getByTestId("search-btn"));

    expect(screen.getByTestId("search-btn")).toHaveTextContent(/searching/i);

    // Resolve to clean up
    resolveSearch?.({
      ok: true,
      json: async () => ({ data: [] }),
    });
  });

  it("fetches owned numbers on mount", async () => {
    setupFetch({ owned: ownedNumbers });
    render(<PhoneNumberManager org="org1" />);

    await waitFor(() => {
      const calls = mockFetch.mock.calls;
      const getCall = calls.find(
        (c: unknown[]) =>
          typeof c[0] === "string" &&
          c[0].includes("/channels/voice/numbers") &&
          !c[0].includes("/search") &&
          !c[0].includes("/purchase"),
      );
      expect(getCall).toBeDefined();
    });
  });

  it("uses correct org in API URL", async () => {
    setupFetch({ owned: [] });
    render(<PhoneNumberManager org="my-org" />);

    await waitFor(() => {
      const calls = mockFetch.mock.calls;
      const getCall = calls.find(
        (c: unknown[]) => typeof c[0] === "string" && c[0].includes("/orgs/my-org/"),
      );
      expect(getCall).toBeDefined();
    });
  });

  it("renders table column headers for owned numbers", async () => {
    setupFetch({ owned: ownedNumbers });
    render(<PhoneNumberManager org="org1" />);
    await screen.findByText("+15551234567");

    const table = screen.getByTestId("owned-numbers-table");
    expect(within(table).getByText("Number")).toBeInTheDocument();
    expect(within(table).getByText("Status")).toBeInTheDocument();
    expect(within(table).getByText("Dispatch Rule")).toBeInTheDocument();
    expect(within(table).getByText("Purchased")).toBeInTheDocument();
  });
});
