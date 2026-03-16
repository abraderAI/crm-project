import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { SecurityLog } from "./security-log";
import type { SecurityLogEntry } from "@/lib/api-types";

/* ---------- fixtures ---------- */

const entry1: SecurityLogEntry = {
  id: "sl-1",
  user_id: "user-abc",
  ip_address: "10.0.0.1",
  user_agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
  timestamp: "2026-03-15T14:30:00Z",
};

const entry2: SecurityLogEntry = {
  id: "sl-2",
  user_id: "user-def",
  ip_address: "192.168.1.42",
  user_agent: "curl/8.1.2",
  timestamp: "2026-03-14T09:15:00Z",
};

const entry3: SecurityLogEntry = {
  id: "sl-3",
  user_id: "user-ghi",
  ip_address: "172.16.0.5",
  user_agent: "PostmanRuntime/7.36.0",
  timestamp: "2026-03-13T22:00:00Z",
};

/* ---------- mocks ---------- */

vi.mock("next/link", () => ({
  __esModule: true,
  default: ({
    href,
    children,
    ...rest
  }: {
    href: string;
    children: React.ReactNode;
    [key: string]: unknown;
  }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}));

/* ---------- tests ---------- */

describe("SecurityLog", () => {
  /* --- empty / loading states --- */

  it("shows empty state when no entries provided", () => {
    render(<SecurityLog entries={[]} />);
    expect(screen.getByTestId("security-log-empty")).toHaveTextContent("No entries found.");
  });

  it("shows loading state", () => {
    render(<SecurityLog entries={[]} loading={true} />);
    expect(screen.getByTestId("security-log-loading")).toBeInTheDocument();
  });

  it("does not show empty state while loading", () => {
    render(<SecurityLog entries={[]} loading={true} />);
    expect(screen.queryByTestId("security-log-empty")).not.toBeInTheDocument();
  });

  /* --- table rendering --- */

  it("renders table headers", () => {
    render(<SecurityLog entries={[entry1]} />);
    expect(screen.getByText("User ID")).toBeInTheDocument();
    expect(screen.getByText("IP Address")).toBeInTheDocument();
    expect(screen.getByText("User Agent")).toBeInTheDocument();
    expect(screen.getByText("Timestamp")).toBeInTheDocument();
  });

  it("renders entry rows", () => {
    render(<SecurityLog entries={[entry1, entry2]} />);
    expect(screen.getByTestId("security-row-sl-1")).toBeInTheDocument();
    expect(screen.getByTestId("security-row-sl-2")).toBeInTheDocument();
  });

  it("renders all three entries", () => {
    render(<SecurityLog entries={[entry1, entry2, entry3]} />);
    expect(screen.getByTestId("security-row-sl-1")).toBeInTheDocument();
    expect(screen.getByTestId("security-row-sl-2")).toBeInTheDocument();
    expect(screen.getByTestId("security-row-sl-3")).toBeInTheDocument();
  });

  /* --- cell content --- */

  it("displays user ID as link to user detail", () => {
    render(<SecurityLog entries={[entry1]} />);
    const link = screen.getByTestId("security-user-sl-1");
    expect(link).toHaveTextContent("user-abc");
    expect(link.closest("a")).toHaveAttribute("href", "/admin/users/user-abc");
  });

  it("displays IP address", () => {
    render(<SecurityLog entries={[entry1]} />);
    expect(screen.getByTestId("security-ip-sl-1")).toHaveTextContent("10.0.0.1");
  });

  it("displays user agent", () => {
    render(<SecurityLog entries={[entry1]} />);
    expect(screen.getByTestId("security-ua-sl-1")).toHaveTextContent(
      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
    );
  });

  it("displays formatted timestamp", () => {
    render(<SecurityLog entries={[entry1]} />);
    expect(screen.getByTestId("security-time-sl-1")).toBeInTheDocument();
    /* We verify the element exists and has non-empty text; exact format depends on locale. */
    expect(screen.getByTestId("security-time-sl-1").textContent?.length).toBeGreaterThan(0);
  });

  /* --- pagination --- */

  it("renders load more button when hasMore is true", () => {
    render(<SecurityLog entries={[entry1]} hasMore={true} onLoadMore={vi.fn()} />);
    expect(screen.getByTestId("security-log-load-more")).toBeInTheDocument();
  });

  it("does not render load more button when hasMore is false", () => {
    render(<SecurityLog entries={[entry1]} hasMore={false} />);
    expect(screen.queryByTestId("security-log-load-more")).not.toBeInTheDocument();
  });

  it("calls onLoadMore when load more button is clicked", async () => {
    const user = userEvent.setup();
    const onLoadMore = vi.fn();
    render(<SecurityLog entries={[entry1]} hasMore={true} onLoadMore={onLoadMore} />);

    await user.click(screen.getByTestId("security-log-load-more"));
    expect(onLoadMore).toHaveBeenCalledOnce();
  });

  it("does not render load more while loading", () => {
    render(<SecurityLog entries={[entry1]} hasMore={true} loading={true} onLoadMore={vi.fn()} />);
    expect(screen.queryByTestId("security-log-load-more")).not.toBeInTheDocument();
  });

  /* --- truncation of user agent --- */

  it("truncates long user agents with title tooltip", () => {
    const longUA =
      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36";
    const longEntry: SecurityLogEntry = { ...entry1, user_agent: longUA };
    render(<SecurityLog entries={[longEntry]} />);
    const el = screen.getByTestId("security-ua-sl-1");
    expect(el).toHaveAttribute("title", longUA);
  });
});
