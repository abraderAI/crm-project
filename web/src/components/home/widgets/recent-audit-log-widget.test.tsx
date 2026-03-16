import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { RecentAuditLogWidget } from "./recent-audit-log-widget";

const mockFetchRecentAuditEvents = vi.fn();

vi.mock("@/lib/widget-api", () => ({
  fetchRecentAuditEvents: (...args: unknown[]) => mockFetchRecentAuditEvents(...args),
}));

const sampleEvents = [
  {
    id: "a1",
    actor: "admin-1",
    action: "create",
    entity_type: "org",
    entity_id: "o1",
    created_at: "2026-01-01",
  },
  {
    id: "a2",
    actor: "user-2",
    action: "update",
    entity_type: "thread",
    entity_id: "t1",
    created_at: "2026-01-02",
  },
  {
    id: "a3",
    actor: "user-3",
    action: "delete",
    entity_type: "user",
    entity_id: "u1",
    created_at: "2026-01-03",
  },
];

describe("RecentAuditLogWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state initially", () => {
    mockFetchRecentAuditEvents.mockReturnValue(new Promise(() => {}));
    render(<RecentAuditLogWidget token="tok" />);
    expect(screen.getByTestId("audit-log-loading")).toBeInTheDocument();
  });

  it("renders audit events after loading", async () => {
    mockFetchRecentAuditEvents.mockResolvedValue(sampleEvents);

    render(<RecentAuditLogWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("audit-log-content")).toBeInTheDocument();
    });

    expect(screen.getByTestId("audit-a1")).toBeInTheDocument();
    expect(screen.getByTestId("audit-a2")).toBeInTheDocument();
    expect(screen.getByText("admin-1")).toBeInTheDocument();
    expect(screen.getByTestId("audit-action-a1")).toHaveTextContent("create");
    expect(screen.getByTestId("audit-action-a3")).toHaveTextContent("delete");
  });

  it("displays entity info on each event", async () => {
    mockFetchRecentAuditEvents.mockResolvedValue(sampleEvents);

    render(<RecentAuditLogWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByText("org/o1")).toBeInTheDocument();
      expect(screen.getByText("thread/t1")).toBeInTheDocument();
    });
  });

  it("shows error state on failure", async () => {
    mockFetchRecentAuditEvents.mockRejectedValue(new Error("fail"));

    render(<RecentAuditLogWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("audit-log-error")).toBeInTheDocument();
    });
  });

  it("shows empty state when no events", async () => {
    mockFetchRecentAuditEvents.mockResolvedValue([]);

    render(<RecentAuditLogWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("audit-log-empty")).toBeInTheDocument();
    });
  });

  it("passes token and limit to API", async () => {
    mockFetchRecentAuditEvents.mockResolvedValue([]);

    render(<RecentAuditLogWidget token="my-token" />);

    await waitFor(() => {
      expect(mockFetchRecentAuditEvents).toHaveBeenCalledWith("my-token", 10);
    });
  });
});
