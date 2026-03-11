import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ModerationQueue } from "./moderation-queue";
import type { Flag } from "@/lib/api-types";

const pendingFlag: Flag = {
  id: "f1",
  thread_id: "t1",
  reporter_id: "user-1",
  reason: "Spam content",
  status: "pending",
  created_at: "2026-01-15T10:00:00Z",
  updated_at: "2026-01-15T10:00:00Z",
};

const resolvedFlag: Flag = {
  id: "f2",
  thread_id: "t2",
  reporter_id: "user-2",
  reason: "Harassment",
  status: "resolved",
  resolved_by: "mod-1",
  resolution_note: "Action taken",
  created_at: "2026-01-14T10:00:00Z",
  updated_at: "2026-01-15T10:00:00Z",
};

const dismissedFlag: Flag = {
  id: "f3",
  thread_id: "t3",
  reporter_id: "user-3",
  reason: "False report",
  status: "dismissed",
  created_at: "2026-01-13T10:00:00Z",
  updated_at: "2026-01-14T10:00:00Z",
};

describe("ModerationQueue", () => {
  it("renders the queue heading", () => {
    render(<ModerationQueue flags={[]} onResolve={vi.fn()} onDismiss={vi.fn()} />);
    expect(screen.getByText("Moderation Queue")).toBeInTheDocument();
  });

  it("renders the moderation icon", () => {
    render(<ModerationQueue flags={[]} onResolve={vi.fn()} onDismiss={vi.fn()} />);
    expect(screen.getByTestId("moderation-icon")).toBeInTheDocument();
  });

  it("shows the flag count", () => {
    render(
      <ModerationQueue
        flags={[pendingFlag, resolvedFlag]}
        onResolve={vi.fn()}
        onDismiss={vi.fn()}
      />,
    );
    expect(screen.getByTestId("flag-count")).toHaveTextContent("2");
  });

  it("shows empty state when no flags", () => {
    render(<ModerationQueue flags={[]} onResolve={vi.fn()} onDismiss={vi.fn()} />);
    expect(screen.getByTestId("moderation-empty")).toHaveTextContent("No flags to review.");
  });

  it("shows loading state", () => {
    render(<ModerationQueue flags={[]} loading={true} onResolve={vi.fn()} onDismiss={vi.fn()} />);
    expect(screen.getByTestId("moderation-loading")).toHaveTextContent("Loading flags...");
  });

  it("renders flag items", () => {
    render(
      <ModerationQueue
        flags={[pendingFlag, resolvedFlag]}
        onResolve={vi.fn()}
        onDismiss={vi.fn()}
      />,
    );
    expect(screen.getByTestId("flag-item-f1")).toBeInTheDocument();
    expect(screen.getByTestId("flag-item-f2")).toBeInTheDocument();
  });

  it("displays flag reason", () => {
    render(<ModerationQueue flags={[pendingFlag]} onResolve={vi.fn()} onDismiss={vi.fn()} />);
    expect(screen.getByTestId("flag-reason-f1")).toHaveTextContent("Spam content");
  });

  it("displays flag status", () => {
    render(<ModerationQueue flags={[pendingFlag]} onResolve={vi.fn()} onDismiss={vi.fn()} />);
    expect(screen.getByTestId("flag-status-f1")).toHaveTextContent("pending");
  });

  it("displays reporter ID", () => {
    render(<ModerationQueue flags={[pendingFlag]} onResolve={vi.fn()} onDismiss={vi.fn()} />);
    expect(screen.getByTestId("flag-reporter-f1")).toHaveTextContent("Reporter: user-1");
  });

  it("shows resolve/dismiss actions for pending flags", () => {
    render(<ModerationQueue flags={[pendingFlag]} onResolve={vi.fn()} onDismiss={vi.fn()} />);
    expect(screen.getByTestId("flag-actions-f1")).toBeInTheDocument();
    expect(screen.getByTestId("flag-resolve-f1")).toBeInTheDocument();
    expect(screen.getByTestId("flag-dismiss-f1")).toBeInTheDocument();
  });

  it("hides actions for resolved flags", () => {
    render(<ModerationQueue flags={[resolvedFlag]} onResolve={vi.fn()} onDismiss={vi.fn()} />);
    expect(screen.queryByTestId("flag-actions-f2")).not.toBeInTheDocument();
  });

  it("hides actions for dismissed flags", () => {
    render(<ModerationQueue flags={[dismissedFlag]} onResolve={vi.fn()} onDismiss={vi.fn()} />);
    expect(screen.queryByTestId("flag-actions-f3")).not.toBeInTheDocument();
  });

  it("calls onResolve when resolve clicked", async () => {
    const user = userEvent.setup();
    const onResolve = vi.fn();
    render(<ModerationQueue flags={[pendingFlag]} onResolve={onResolve} onDismiss={vi.fn()} />);

    await user.click(screen.getByTestId("flag-resolve-f1"));
    expect(onResolve).toHaveBeenCalledWith("f1", "");
  });

  it("calls onDismiss when dismiss clicked", async () => {
    const user = userEvent.setup();
    const onDismiss = vi.fn();
    render(<ModerationQueue flags={[pendingFlag]} onResolve={vi.fn()} onDismiss={onDismiss} />);

    await user.click(screen.getByTestId("flag-dismiss-f1"));
    expect(onDismiss).toHaveBeenCalledWith("f1");
  });

  it("shows resolution note for resolved flags", () => {
    render(<ModerationQueue flags={[resolvedFlag]} onResolve={vi.fn()} onDismiss={vi.fn()} />);
    expect(screen.getByTestId("flag-note-f2")).toHaveTextContent("Note: Action taken");
  });

  it("renders filter buttons when onFilterChange provided", () => {
    render(
      <ModerationQueue
        flags={[pendingFlag]}
        onResolve={vi.fn()}
        onDismiss={vi.fn()}
        onFilterChange={vi.fn()}
      />,
    );
    expect(screen.getByTestId("moderation-filters")).toBeInTheDocument();
    expect(screen.getByTestId("filter-all")).toBeInTheDocument();
    expect(screen.getByTestId("filter-pending")).toBeInTheDocument();
  });

  it("calls onFilterChange when filter clicked", async () => {
    const user = userEvent.setup();
    const onFilterChange = vi.fn();
    render(
      <ModerationQueue
        flags={[pendingFlag]}
        onResolve={vi.fn()}
        onDismiss={vi.fn()}
        onFilterChange={onFilterChange}
      />,
    );

    await user.click(screen.getByTestId("filter-pending"));
    expect(onFilterChange).toHaveBeenCalledWith("pending");
  });

  it("highlights active filter", () => {
    render(
      <ModerationQueue
        flags={[pendingFlag]}
        onResolve={vi.fn()}
        onDismiss={vi.fn()}
        statusFilter="pending"
        onFilterChange={vi.fn()}
      />,
    );
    expect(screen.getByTestId("filter-pending")).toHaveClass("bg-primary");
  });

  it("displays formatted date", () => {
    render(<ModerationQueue flags={[pendingFlag]} onResolve={vi.fn()} onDismiss={vi.fn()} />);
    expect(screen.getByTestId("flag-date-f1")).toBeInTheDocument();
  });
});
