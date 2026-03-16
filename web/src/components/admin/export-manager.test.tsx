import { render, screen, act, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import type { AdminExport } from "@/lib/api-types";

// Mock Clerk auth.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

// Mock api-client.
const mockClientMutate = vi.fn();
const mockBuildUrl = vi.fn((path: string) => `http://localhost:8080/v1${path}`);
const mockBuildHeaders = vi.fn((token: string) => ({ Authorization: `Bearer ${token}` }));
const mockParseResponse = vi.fn();
vi.mock("@/lib/api-client", () => ({
  clientMutate: (...args: unknown[]) => mockClientMutate(...args),
  buildUrl: (...args: unknown[]) => mockBuildUrl(...args),
  buildHeaders: (...args: unknown[]) => mockBuildHeaders(...args),
  parseResponse: (...args: unknown[]) => mockParseResponse(...args),
}));

// Mock global fetch for polling GET requests.
const mockFetch = vi.fn();

// Import after mocks.
import { ExportManager } from "./export-manager";

const pendingExport: AdminExport = {
  id: "exp-1",
  type: "users",
  filters: "{}",
  format: "csv",
  status: "pending",
  requested_by: "user-1",
  created_at: "2026-03-15T10:00:00Z",
};

const completedExport: AdminExport = {
  id: "exp-2",
  type: "orgs",
  filters: "{}",
  format: "json",
  status: "completed",
  file_path: "export-orgs-exp-2.json",
  requested_by: "user-1",
  created_at: "2026-03-14T08:00:00Z",
  completed_at: "2026-03-14T08:05:00Z",
};

const failedExport: AdminExport = {
  id: "exp-3",
  type: "audit",
  filters: "{}",
  format: "csv",
  status: "failed",
  requested_by: "user-1",
  error_msg: "query timeout",
  created_at: "2026-03-13T12:00:00Z",
};

const processingExport: AdminExport = {
  id: "exp-4",
  type: "users",
  filters: "{}",
  format: "json",
  status: "processing",
  requested_by: "user-1",
  created_at: "2026-03-15T11:00:00Z",
};

describe("ExportManager", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers({ shouldAdvanceTime: true });
    mockGetToken.mockResolvedValue("test-token");
    // Default: stub fetch for polling (parseResponse extracts the value).
    mockFetch.mockResolvedValue(new Response("{}"));
    vi.stubGlobal("fetch", mockFetch);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders the component with heading", () => {
    render(<ExportManager initialExports={[]} />);
    expect(screen.getByTestId("export-manager")).toBeInTheDocument();
    expect(screen.getByText("Data Exports")).toBeInTheDocument();
  });

  it("renders the trigger form with type and format selectors", () => {
    render(<ExportManager initialExports={[]} />);
    expect(screen.getByTestId("export-type-select")).toBeInTheDocument();
    expect(screen.getByTestId("export-format-select")).toBeInTheDocument();
    expect(screen.getByTestId("export-create-btn")).toBeInTheDocument();
  });

  it("renders type selector with all options", () => {
    render(<ExportManager initialExports={[]} />);
    const select = screen.getByTestId("export-type-select");
    const options = within(select).getAllByRole("option");
    const values = options.map((o) => (o as HTMLOptionElement).value);
    expect(values).toContain("users");
    expect(values).toContain("orgs");
    expect(values).toContain("audit");
  });

  it("renders format selector with csv and json", () => {
    render(<ExportManager initialExports={[]} />);
    const select = screen.getByTestId("export-format-select");
    const options = within(select).getAllByRole("option");
    const values = options.map((o) => (o as HTMLOptionElement).value);
    expect(values).toContain("csv");
    expect(values).toContain("json");
  });

  it("shows empty state when no exports", () => {
    render(<ExportManager initialExports={[]} />);
    expect(screen.getByTestId("export-empty")).toBeInTheDocument();
  });

  it("renders export history table with exports", () => {
    render(<ExportManager initialExports={[completedExport, failedExport]} />);
    expect(screen.getByTestId("export-table")).toBeInTheDocument();
    expect(screen.getByTestId("export-row-exp-2")).toBeInTheDocument();
    expect(screen.getByTestId("export-row-exp-3")).toBeInTheDocument();
  });

  it("displays export type and format in table rows", () => {
    render(<ExportManager initialExports={[completedExport]} />);
    const row = screen.getByTestId("export-row-exp-2");
    expect(within(row).getByTestId("export-type-exp-2")).toHaveTextContent("orgs");
    expect(within(row).getByTestId("export-format-exp-2")).toHaveTextContent("json");
  });

  it("displays status badge for each export", () => {
    render(<ExportManager initialExports={[pendingExport, completedExport, failedExport]} />);
    expect(screen.getByTestId("export-status-exp-1")).toHaveTextContent("pending");
    expect(screen.getByTestId("export-status-exp-2")).toHaveTextContent("completed");
    expect(screen.getByTestId("export-status-exp-3")).toHaveTextContent("failed");
  });

  it("shows download button only for completed exports", () => {
    render(<ExportManager initialExports={[pendingExport, completedExport, failedExport]} />);
    expect(screen.queryByTestId("export-download-exp-1")).not.toBeInTheDocument();
    expect(screen.getByTestId("export-download-exp-2")).toBeInTheDocument();
    expect(screen.queryByTestId("export-download-exp-3")).not.toBeInTheDocument();
  });

  it("download button links to correct file path", () => {
    render(<ExportManager initialExports={[completedExport]} />);
    const downloadBtn = screen.getByTestId("export-download-exp-2");
    expect(downloadBtn).toHaveAttribute("href", expect.stringContaining("export-orgs-exp-2.json"));
  });

  it("submits create export form with POST to /admin/exports", async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    mockClientMutate.mockResolvedValue({
      export_id: "exp-new",
      status: "pending",
    });

    render(<ExportManager initialExports={[]} />);

    await user.selectOptions(screen.getByTestId("export-type-select"), "orgs");
    await user.selectOptions(screen.getByTestId("export-format-select"), "json");
    await user.click(screen.getByTestId("export-create-btn"));

    expect(mockClientMutate).toHaveBeenCalledWith("POST", "/admin/exports", {
      token: "test-token",
      body: { type: "orgs", format: "json" },
    });
  });

  it("adds newly created export to the history table", async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    mockClientMutate.mockImplementation((method: string): Promise<unknown> => {
      if (method === "POST") {
        return Promise.resolve({ export_id: "exp-new", status: "pending" });
      }
      // GET for polling
      return Promise.resolve({
        id: "exp-new",
        type: "users",
        filters: "{}",
        format: "csv",
        status: "pending",
        requested_by: "user-1",
        created_at: "2026-03-16T00:00:00Z",
      });
    });

    render(<ExportManager initialExports={[]} />);
    await user.click(screen.getByTestId("export-create-btn"));

    expect(screen.getByTestId("export-row-exp-new")).toBeInTheDocument();
  });

  it("polls pending exports every 3 seconds", async () => {
    mockParseResponse.mockResolvedValue({
      ...pendingExport,
      status: "completed",
      file_path: "export-users.csv",
    });

    render(<ExportManager initialExports={[pendingExport]} />);

    // Initial render — polling should start for pending export.
    await act(async () => {
      vi.advanceTimersByTime(3000);
    });

    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/v1/admin/exports/exp-1",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("updates status badge when poll returns completed", async () => {
    mockParseResponse.mockResolvedValue({
      ...pendingExport,
      status: "completed",
      file_path: "export-users-exp-1.csv",
    });

    render(<ExportManager initialExports={[pendingExport]} />);

    await act(async () => {
      vi.advanceTimersByTime(3000);
    });

    expect(screen.getByTestId("export-status-exp-1")).toHaveTextContent("completed");
  });

  it("shows download button after export completes via polling", async () => {
    mockParseResponse.mockResolvedValue({
      ...pendingExport,
      status: "completed",
      file_path: "export-users-exp-1.csv",
    });

    render(<ExportManager initialExports={[pendingExport]} />);

    expect(screen.queryByTestId("export-download-exp-1")).not.toBeInTheDocument();

    await act(async () => {
      vi.advanceTimersByTime(3000);
    });

    expect(screen.getByTestId("export-download-exp-1")).toBeInTheDocument();
  });

  it("stops polling when all exports are complete", async () => {
    mockParseResponse.mockResolvedValue({
      ...pendingExport,
      status: "completed",
      file_path: "export-users-exp-1.csv",
    });

    render(<ExportManager initialExports={[pendingExport]} />);

    // First poll.
    await act(async () => {
      vi.advanceTimersByTime(3000);
    });

    mockFetch.mockClear();

    // Second tick should not poll again since all are complete.
    await act(async () => {
      vi.advanceTimersByTime(3000);
    });

    expect(mockFetch).not.toHaveBeenCalled();
  });

  it("does not poll when no pending exports exist", async () => {
    render(<ExportManager initialExports={[completedExport]} />);

    await act(async () => {
      vi.advanceTimersByTime(6000);
    });

    expect(mockFetch).not.toHaveBeenCalled();
  });

  it("polls processing exports alongside pending", async () => {
    mockParseResponse.mockResolvedValue({
      ...processingExport,
      status: "completed",
      file_path: "export-users-exp-4.json",
    });

    render(<ExportManager initialExports={[processingExport]} />);

    await act(async () => {
      vi.advanceTimersByTime(3000);
    });

    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/v1/admin/exports/exp-4",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("displays created at date for exports", () => {
    render(<ExportManager initialExports={[completedExport]} />);
    expect(screen.getByTestId("export-date-exp-2")).toBeInTheDocument();
  });

  it("displays export ID in the table", () => {
    render(<ExportManager initialExports={[completedExport]} />);
    expect(screen.getByTestId("export-id-exp-2")).toHaveTextContent("exp-2");
  });

  it("disables create button while submitting", async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    let resolveCreate: (value: unknown) => void;
    mockClientMutate.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveCreate = resolve;
        }),
    );

    render(<ExportManager initialExports={[]} />);
    await user.click(screen.getByTestId("export-create-btn"));

    expect(screen.getByTestId("export-create-btn")).toBeDisabled();

    await act(async () => {
      resolveCreate!({ export_id: "exp-x", status: "pending" });
    });

    expect(screen.getByTestId("export-create-btn")).toBeEnabled();
  });

  it("applies correct color for pending status badge", () => {
    render(<ExportManager initialExports={[pendingExport]} />);
    expect(screen.getByTestId("export-status-exp-1")).toHaveClass("bg-yellow-100");
  });

  it("applies correct color for completed status badge", () => {
    render(<ExportManager initialExports={[completedExport]} />);
    expect(screen.getByTestId("export-status-exp-2")).toHaveClass("bg-green-100");
  });

  it("applies correct color for failed status badge", () => {
    render(<ExportManager initialExports={[failedExport]} />);
    expect(screen.getByTestId("export-status-exp-3")).toHaveClass("bg-red-100");
  });

  it("applies correct color for processing status badge", () => {
    render(<ExportManager initialExports={[processingExport]} />);
    expect(screen.getByTestId("export-status-exp-4")).toHaveClass("bg-blue-100");
  });
});
