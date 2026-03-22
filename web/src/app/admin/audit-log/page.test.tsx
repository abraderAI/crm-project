import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";

// Mock next/navigation.
const mockReplace = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: mockReplace }),
  useSearchParams: () => new URLSearchParams(),
}));

// Mock Clerk auth.
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: vi.fn().mockResolvedValue("tok") }),
}));

// Mock api-client.
vi.mock("@/lib/api-client", () => ({
  buildHeaders: vi.fn().mockReturnValue({}),
  buildUrl: vi.fn().mockImplementation((path: string) => `http://localhost:8080/v1${path}`),
  parseResponse: vi.fn().mockResolvedValue({ data: [], page_info: { has_more: false } }),
}));

// Mock AuditLogViewerWithDirectory.
vi.mock("@/components/admin/audit-log-viewer-wrapper", () => ({
  AuditLogViewerWithDirectory: (props: Record<string, unknown>) => (
    <div data-testid="mock-audit-viewer" data-loading={String(props.loading)} />
  ),
}));

// Mock DateRangePicker.
vi.mock("@/components/reports/date-range-picker", () => ({
  DateRangePicker: () => <div data-testid="mock-date-range-picker" />,
}));

// Stub global fetch.
const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => ({ data: [], page_info: { has_more: false } }) });
vi.stubGlobal("fetch", mockFetch);

import AdminAuditLogPage from "./page";

describe("AdminAuditLogPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockFetch.mockResolvedValue({ ok: true, status: 200 });
  });

  it("renders the audit log container", () => {
    render(<AdminAuditLogPage />);
    expect(screen.getByTestId("admin-audit-log")).toBeInTheDocument();
  });

  it("renders the filter bar", () => {
    render(<AdminAuditLogPage />);
    expect(screen.getByTestId("audit-filter-bar")).toBeInTheDocument();
  });

  it("renders date range picker", () => {
    render(<AdminAuditLogPage />);
    expect(screen.getByTestId("mock-date-range-picker")).toBeInTheDocument();
  });

  it("renders action filter dropdown", () => {
    render(<AdminAuditLogPage />);
    const select = screen.getByTestId("audit-filter-action");
    expect(select).toBeInTheDocument();
    expect(select).toHaveValue("");
  });

  it("renders entity type filter dropdown", () => {
    render(<AdminAuditLogPage />);
    const select = screen.getByTestId("audit-filter-entity-type");
    expect(select).toBeInTheDocument();
    expect(select).toHaveValue("");
  });

  it("renders user search input", () => {
    render(<AdminAuditLogPage />);
    const input = screen.getByTestId("audit-filter-user");
    expect(input).toBeInTheDocument();
    expect(input).toHaveAttribute("placeholder", "Filter by user ID…");
  });

  it("renders the audit log viewer", () => {
    render(<AdminAuditLogPage />);
    expect(screen.getByTestId("mock-audit-viewer")).toBeInTheDocument();
  });

  it("action dropdown has correct options", () => {
    render(<AdminAuditLogPage />);
    const select = screen.getByTestId("audit-filter-action");
    const options = select.querySelectorAll("option");
    expect(options).toHaveLength(4);
    expect(options[0]).toHaveTextContent("All Actions");
    expect(options[1]).toHaveTextContent("Create");
    expect(options[2]).toHaveTextContent("Update");
    expect(options[3]).toHaveTextContent("Delete");
  });

  it("entity type dropdown has correct options", () => {
    render(<AdminAuditLogPage />);
    const select = screen.getByTestId("audit-filter-entity-type");
    const options = select.querySelectorAll("option");
    expect(options.length).toBeGreaterThanOrEqual(7);
    expect(options[0]).toHaveTextContent("All Types");
  });
});
