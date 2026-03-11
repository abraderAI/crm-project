import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { AuditLogViewer } from "./audit-log-viewer";
import type { AuditEntry } from "@/lib/api-types";

const createEntry: AuditEntry = {
  id: "a1",
  user_id: "user-1",
  action: "create",
  entity_type: "thread",
  entity_id: "12345678-abcd",
  before_state: undefined,
  after_state: '{"title":"New Thread"}',
  ip_address: "192.168.1.1",
  request_id: "req-abc",
  created_at: "2026-01-15T10:00:00Z",
  updated_at: "2026-01-15T10:00:00Z",
};

const updateEntry: AuditEntry = {
  id: "a2",
  user_id: "user-2",
  action: "update",
  entity_type: "message",
  entity_id: "87654321-wxyz",
  before_state: '{"body":"old"}',
  after_state: '{"body":"new"}',
  created_at: "2026-01-14T10:00:00Z",
  updated_at: "2026-01-14T10:00:00Z",
};

const deleteEntry: AuditEntry = {
  id: "a3",
  user_id: "user-3",
  action: "delete",
  entity_type: "org",
  entity_id: "abcdefgh-1234",
  created_at: "2026-01-13T10:00:00Z",
  updated_at: "2026-01-13T10:00:00Z",
};

describe("AuditLogViewer", () => {
  it("renders the heading and icon", () => {
    render(<AuditLogViewer entries={[]} />);
    expect(screen.getByText("Audit Log")).toBeInTheDocument();
    expect(screen.getByTestId("audit-icon")).toBeInTheDocument();
  });

  it("shows empty state", () => {
    render(<AuditLogViewer entries={[]} />);
    expect(screen.getByTestId("audit-empty")).toHaveTextContent("No audit log entries.");
  });

  it("shows loading state", () => {
    render(<AuditLogViewer entries={[]} loading={true} />);
    expect(screen.getByTestId("audit-loading")).toBeInTheDocument();
  });

  it("renders audit entries", () => {
    render(<AuditLogViewer entries={[createEntry, updateEntry]} />);
    expect(screen.getByTestId("audit-item-a1")).toBeInTheDocument();
    expect(screen.getByTestId("audit-item-a2")).toBeInTheDocument();
  });

  it("displays action badge", () => {
    render(<AuditLogViewer entries={[createEntry]} />);
    expect(screen.getByTestId("audit-action-a1")).toHaveTextContent("create");
  });

  it("applies green color for create action", () => {
    render(<AuditLogViewer entries={[createEntry]} />);
    expect(screen.getByTestId("audit-action-a1")).toHaveClass("bg-green-100");
  });

  it("applies blue color for update action", () => {
    render(<AuditLogViewer entries={[updateEntry]} />);
    expect(screen.getByTestId("audit-action-a2")).toHaveClass("bg-blue-100");
  });

  it("applies red color for delete action", () => {
    render(<AuditLogViewer entries={[deleteEntry]} />);
    expect(screen.getByTestId("audit-action-a3")).toHaveClass("bg-red-100");
  });

  it("displays entity type and truncated ID", () => {
    render(<AuditLogViewer entries={[createEntry]} />);
    expect(screen.getByTestId("audit-entity-a1")).toHaveTextContent("thread:12345678");
  });

  it("displays user ID", () => {
    render(<AuditLogViewer entries={[createEntry]} />);
    expect(screen.getByTestId("audit-user-a1")).toHaveTextContent("user-1");
  });

  it("shows expand button when before/after state exists", () => {
    render(<AuditLogViewer entries={[createEntry]} />);
    expect(screen.getByTestId("audit-expand-a1")).toBeInTheDocument();
  });

  it("hides expand button when no state data", () => {
    render(<AuditLogViewer entries={[deleteEntry]} />);
    expect(screen.queryByTestId("audit-expand-a3")).not.toBeInTheDocument();
  });

  it("expands diff view on click", async () => {
    const user = userEvent.setup();
    render(<AuditLogViewer entries={[updateEntry]} />);

    expect(screen.queryByTestId("audit-diff-a2")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("audit-expand-a2"));
    expect(screen.getByTestId("audit-diff-a2")).toBeInTheDocument();
  });

  it("shows before/after state in diff view", async () => {
    const user = userEvent.setup();
    render(<AuditLogViewer entries={[updateEntry]} />);

    await user.click(screen.getByTestId("audit-expand-a2"));
    expect(screen.getByTestId("audit-before-a2")).toBeInTheDocument();
    expect(screen.getByTestId("audit-after-a2")).toBeInTheDocument();
  });

  it("displays dash for missing before state", async () => {
    const user = userEvent.setup();
    render(<AuditLogViewer entries={[createEntry]} />);

    await user.click(screen.getByTestId("audit-expand-a1"));
    expect(screen.getByTestId("audit-before-a1")).toHaveTextContent("—");
  });

  it("pretty-prints JSON in diff", async () => {
    const user = userEvent.setup();
    render(<AuditLogViewer entries={[updateEntry]} />);

    await user.click(screen.getByTestId("audit-expand-a2"));
    const before = screen.getByTestId("audit-before-a2");
    expect(before.textContent).toContain('"body"');
  });

  it("collapses diff on second click", async () => {
    const user = userEvent.setup();
    render(<AuditLogViewer entries={[updateEntry]} />);

    await user.click(screen.getByTestId("audit-expand-a2"));
    expect(screen.getByTestId("audit-diff-a2")).toBeInTheDocument();

    await user.click(screen.getByTestId("audit-expand-a2"));
    expect(screen.queryByTestId("audit-diff-a2")).not.toBeInTheDocument();
  });

  it("shows IP address when expanded", async () => {
    const user = userEvent.setup();
    render(<AuditLogViewer entries={[createEntry]} />);

    await user.click(screen.getByTestId("audit-expand-a1"));
    expect(screen.getByTestId("audit-ip-a1")).toHaveTextContent("IP: 192.168.1.1");
  });

  it("shows request ID when expanded", async () => {
    const user = userEvent.setup();
    render(<AuditLogViewer entries={[createEntry]} />);

    await user.click(screen.getByTestId("audit-expand-a1"));
    expect(screen.getByTestId("audit-request-a1")).toHaveTextContent("Request: req-abc");
  });

  it("renders load more button when hasMore", () => {
    render(<AuditLogViewer entries={[createEntry]} hasMore={true} onLoadMore={vi.fn()} />);
    expect(screen.getByTestId("audit-load-more")).toBeInTheDocument();
  });

  it("calls onLoadMore when clicked", async () => {
    const user = userEvent.setup();
    const onLoadMore = vi.fn();
    render(<AuditLogViewer entries={[createEntry]} hasMore={true} onLoadMore={onLoadMore} />);

    await user.click(screen.getByTestId("audit-load-more"));
    expect(onLoadMore).toHaveBeenCalledOnce();
  });

  it("displays formatted date", () => {
    render(<AuditLogViewer entries={[createEntry]} />);
    expect(screen.getByTestId("audit-date-a1")).toBeInTheDocument();
  });
});
