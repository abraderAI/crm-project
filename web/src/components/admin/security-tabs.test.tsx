import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { SecurityTabs } from "./security-tabs";
import type { SecurityLogEntry } from "@/lib/api-types";

/* ---------- fixtures ---------- */

const login1: SecurityLogEntry = {
  id: "login-1",
  user_id: "user-abc",
  ip_address: "10.0.0.1",
  user_agent: "Mozilla/5.0",
  timestamp: "2026-03-15T14:30:00Z",
};

const login2: SecurityLogEntry = {
  id: "login-2",
  user_id: "user-def",
  ip_address: "192.168.1.42",
  user_agent: "curl/8.1.2",
  timestamp: "2026-03-14T09:15:00Z",
};

const failed1: SecurityLogEntry = {
  id: "failed-1",
  user_id: "user-ghi",
  ip_address: "172.16.0.5",
  user_agent: "PostmanRuntime/7.36.0",
  timestamp: "2026-03-13T22:00:00Z",
};

/* ---------- mocks ---------- */

vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: vi.fn().mockResolvedValue(null) }),
}));

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

describe("SecurityTabs", () => {
  /* --- structure --- */

  it("renders the page heading with shield icon", () => {
    render(
      <SecurityTabs
        recentLogins={[]}
        failedAuths={[]}
        loginsHasMore={false}
        failedHasMore={false}
      />,
    );
    expect(screen.getByText("Security Monitoring")).toBeInTheDocument();
    expect(screen.getByTestId("security-icon")).toBeInTheDocument();
  });

  it("renders both tab buttons", () => {
    render(
      <SecurityTabs
        recentLogins={[]}
        failedAuths={[]}
        loginsHasMore={false}
        failedHasMore={false}
      />,
    );
    expect(screen.getByTestId("tab-recent-logins")).toBeInTheDocument();
    expect(screen.getByTestId("tab-failed-auths")).toBeInTheDocument();
  });

  /* --- default tab --- */

  it("defaults to Recent Logins tab as active", () => {
    render(
      <SecurityTabs
        recentLogins={[login1]}
        failedAuths={[failed1]}
        loginsHasMore={false}
        failedHasMore={false}
      />,
    );
    expect(screen.getByTestId("tab-recent-logins")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("tab-failed-auths")).toHaveAttribute("aria-selected", "false");
  });

  it("shows recent logins data by default", () => {
    render(
      <SecurityTabs
        recentLogins={[login1, login2]}
        failedAuths={[failed1]}
        loginsHasMore={false}
        failedHasMore={false}
      />,
    );
    expect(screen.getByTestId("security-row-login-1")).toBeInTheDocument();
    expect(screen.getByTestId("security-row-login-2")).toBeInTheDocument();
    expect(screen.queryByTestId("security-row-failed-1")).not.toBeInTheDocument();
  });

  /* --- tab switching --- */

  it("switches to Failed Auths tab on click", async () => {
    const user = userEvent.setup();
    render(
      <SecurityTabs
        recentLogins={[login1]}
        failedAuths={[failed1]}
        loginsHasMore={false}
        failedHasMore={false}
      />,
    );

    await user.click(screen.getByTestId("tab-failed-auths"));
    expect(screen.getByTestId("tab-failed-auths")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("tab-recent-logins")).toHaveAttribute("aria-selected", "false");
    expect(screen.getByTestId("security-row-failed-1")).toBeInTheDocument();
    expect(screen.queryByTestId("security-row-login-1")).not.toBeInTheDocument();
  });

  it("switches back to Recent Logins tab", async () => {
    const user = userEvent.setup();
    render(
      <SecurityTabs
        recentLogins={[login1]}
        failedAuths={[failed1]}
        loginsHasMore={false}
        failedHasMore={false}
      />,
    );

    await user.click(screen.getByTestId("tab-failed-auths"));
    await user.click(screen.getByTestId("tab-recent-logins"));
    expect(screen.getByTestId("tab-recent-logins")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("security-row-login-1")).toBeInTheDocument();
  });

  /* --- empty states per tab --- */

  it("shows empty state for Recent Logins when no logins", () => {
    render(
      <SecurityTabs
        recentLogins={[]}
        failedAuths={[failed1]}
        loginsHasMore={false}
        failedHasMore={false}
      />,
    );
    expect(screen.getByTestId("security-log-empty")).toBeInTheDocument();
  });

  it("shows empty state for Failed Auths when no failures", async () => {
    const user = userEvent.setup();
    render(
      <SecurityTabs
        recentLogins={[login1]}
        failedAuths={[]}
        loginsHasMore={false}
        failedHasMore={false}
      />,
    );

    await user.click(screen.getByTestId("tab-failed-auths"));
    expect(screen.getByTestId("security-log-empty")).toBeInTheDocument();
  });

  /* --- pagination per tab --- */

  it("passes loginsHasMore to Recent Logins tab", () => {
    render(
      <SecurityTabs
        recentLogins={[login1]}
        failedAuths={[]}
        loginsHasMore={true}
        failedHasMore={false}
      />,
    );
    expect(screen.getByTestId("security-log-load-more")).toBeInTheDocument();
  });

  it("passes failedHasMore to Failed Auths tab", async () => {
    const user = userEvent.setup();
    render(
      <SecurityTabs
        recentLogins={[]}
        failedAuths={[failed1]}
        loginsHasMore={false}
        failedHasMore={true}
      />,
    );

    await user.click(screen.getByTestId("tab-failed-auths"));
    expect(screen.getByTestId("security-log-load-more")).toBeInTheDocument();
  });
});
