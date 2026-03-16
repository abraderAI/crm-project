import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";
import type { ApiUsageEntry } from "@/lib/api-types";

// Mock Clerk auth.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

// Mock api-client to intercept buildUrl + fetch.
vi.mock("@/lib/api-client", () => ({
  buildUrl: (path: string, params?: Record<string, string>) => {
    const url = new URL(`http://localhost:8080/v1${path}`);
    if (params) {
      for (const [k, v] of Object.entries(params)) {
        url.searchParams.set(k, v);
      }
    }
    return url.toString();
  },
  buildHeaders: (token?: string | null) => {
    const h: Record<string, string> = {
      "Content-Type": "application/json",
      Accept: "application/json",
    };
    if (token) h["Authorization"] = `Bearer ${token}`;
    return h;
  },
}));

import { ApiUsageTable } from "./api-usage-table";

const fixtureEntries: ApiUsageEntry[] = [
  { endpoint: "/v1/orgs", method: "GET", count: 450 },
  { endpoint: "/v1/threads", method: "POST", count: 320 },
  { endpoint: "/v1/users", method: "GET", count: 120 },
];

describe("ApiUsageTable", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");

    // Default fetch mock returning fixture data.
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ period: "24h", data: fixtureEntries }),
      }),
    );
  });

  it("renders loading state initially", () => {
    render(<ApiUsageTable />);
    expect(screen.getByTestId("api-usage-loading")).toBeInTheDocument();
  });

  it("renders table with fixture data after fetch", async () => {
    render(<ApiUsageTable />);
    await waitFor(() => {
      expect(screen.getByTestId("api-usage-table")).toBeInTheDocument();
    });

    expect(screen.getByText("/v1/orgs")).toBeInTheDocument();
    expect(screen.getByText("/v1/threads")).toBeInTheDocument();
    expect(screen.getByText("/v1/users")).toBeInTheDocument();
  });

  it("renders endpoint, method, and count columns", async () => {
    render(<ApiUsageTable />);
    await waitFor(() => {
      expect(screen.getByTestId("api-usage-table")).toBeInTheDocument();
    });

    expect(screen.getByText("Endpoint")).toBeInTheDocument();
    expect(screen.getByText("Method")).toBeInTheDocument();
    expect(screen.getByText("Requests")).toBeInTheDocument();
  });

  it("renders all three time-window toggle buttons", async () => {
    render(<ApiUsageTable />);
    await waitFor(() => {
      expect(screen.getByTestId("api-usage-table")).toBeInTheDocument();
    });

    expect(screen.getByTestId("period-btn-24h")).toBeInTheDocument();
    expect(screen.getByTestId("period-btn-7d")).toBeInTheDocument();
    expect(screen.getByTestId("period-btn-30d")).toBeInTheDocument();
  });

  it("has 24h selected by default", async () => {
    render(<ApiUsageTable />);
    await waitFor(() => {
      expect(screen.getByTestId("api-usage-table")).toBeInTheDocument();
    });

    expect(screen.getByTestId("period-btn-24h")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("period-btn-7d")).toHaveAttribute("aria-pressed", "false");
    expect(screen.getByTestId("period-btn-30d")).toHaveAttribute("aria-pressed", "false");
  });

  it("fetches with period=24h on initial load", async () => {
    render(<ApiUsageTable />);
    await waitFor(() => {
      expect(screen.getByTestId("api-usage-table")).toBeInTheDocument();
    });

    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining("period=24h"),
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("re-fetches with period=7d when 7d toggle is clicked", async () => {
    const user = userEvent.setup();
    render(<ApiUsageTable />);
    await waitFor(() => {
      expect(screen.getByTestId("api-usage-table")).toBeInTheDocument();
    });

    vi.mocked(fetch).mockClear();
    await user.click(screen.getByTestId("period-btn-7d"));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining("period=7d"),
        expect.objectContaining({ method: "GET" }),
      );
    });
  });

  it("re-fetches with period=30d when 30d toggle is clicked", async () => {
    const user = userEvent.setup();
    render(<ApiUsageTable />);
    await waitFor(() => {
      expect(screen.getByTestId("api-usage-table")).toBeInTheDocument();
    });

    vi.mocked(fetch).mockClear();
    await user.click(screen.getByTestId("period-btn-30d"));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining("period=30d"),
        expect.objectContaining({ method: "GET" }),
      );
    });
  });

  it("shows empty state when no data returned", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ period: "24h", data: [] }),
      }),
    );

    render(<ApiUsageTable />);
    await waitFor(() => {
      expect(screen.getByTestId("api-usage-empty")).toBeInTheDocument();
    });
  });

  it("shows error state on fetch failure", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        statusText: "Internal Server Error",
        json: () =>
          Promise.resolve({
            type: "about:blank",
            title: "Error",
            status: 500,
          }),
      }),
    );

    render(<ApiUsageTable />);
    await waitFor(() => {
      expect(screen.getByTestId("api-usage-error")).toBeInTheDocument();
    });
  });

  it("renders rows sorted by count descending", async () => {
    render(<ApiUsageTable />);
    await waitFor(() => {
      expect(screen.getByTestId("api-usage-table")).toBeInTheDocument();
    });

    const rows = screen.getAllByTestId(/^api-usage-row-/);
    expect(rows).toHaveLength(3);
    // Data comes pre-sorted from backend, but verify order in DOM.
    expect(rows[0]).toHaveTextContent("450");
    expect(rows[1]).toHaveTextContent("320");
    expect(rows[2]).toHaveTextContent("120");
  });

  it("renders method badges for each row", async () => {
    render(<ApiUsageTable />);
    await waitFor(() => {
      expect(screen.getByTestId("api-usage-table")).toBeInTheDocument();
    });

    expect(screen.getByTestId("api-usage-method-0")).toHaveTextContent("GET");
    expect(screen.getByTestId("api-usage-method-1")).toHaveTextContent("POST");
  });

  it("does not re-fetch when clicking the already-active period", async () => {
    const user = userEvent.setup();
    render(<ApiUsageTable />);
    await waitFor(() => {
      expect(screen.getByTestId("api-usage-table")).toBeInTheDocument();
    });

    vi.mocked(fetch).mockClear();
    await user.click(screen.getByTestId("period-btn-24h"));

    // Should not have made a new fetch since 24h is already active.
    expect(fetch).not.toHaveBeenCalled();
  });
});
