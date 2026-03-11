import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ThemeProvider } from "@/components/theme-provider";
import { AppLayout } from "./app-layout";
import type { NavItem } from "./sidebar";

const navItems: NavItem[] = [
  {
    id: "org-1",
    label: "Test Org",
    href: "/orgs/test",
    type: "org",
  },
];

function renderLayout(
  props: Partial<Parameters<typeof AppLayout>[0]> = {},
): ReturnType<typeof render> {
  return render(
    <ThemeProvider>
      <AppLayout navItems={navItems} {...props}>
        <div data-testid="content">Main content</div>
      </AppLayout>
    </ThemeProvider>,
  );
}

describe("AppLayout", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("renders app layout container", () => {
    renderLayout();
    expect(screen.getByTestId("app-layout")).toBeInTheDocument();
  });

  it("renders sidebar with nav items", () => {
    renderLayout();
    expect(screen.getByTestId("sidebar")).toBeInTheDocument();
    expect(screen.getByText("Test Org")).toBeInTheDocument();
  });

  it("renders topbar", () => {
    renderLayout();
    expect(screen.getByTestId("topbar")).toBeInTheDocument();
  });

  it("renders children content", () => {
    renderLayout();
    expect(screen.getByTestId("content")).toHaveTextContent("Main content");
  });

  it("renders breadcrumbs when provided", () => {
    renderLayout({ breadcrumbs: [{ label: "Home", href: "/" }, { label: "Page" }] });
    expect(screen.getByTestId("breadcrumbs")).toBeInTheDocument();
  });

  it("does not render breadcrumbs when empty", () => {
    renderLayout({ breadcrumbs: [] });
    expect(screen.queryByTestId("breadcrumbs")).not.toBeInTheDocument();
  });

  it("toggles sidebar on button click", async () => {
    const user = userEvent.setup();
    renderLayout();

    // Initially expanded — nav items visible.
    expect(screen.getByText("Test Org")).toBeInTheDocument();

    // Click collapse.
    await user.click(screen.getByTestId("sidebar-toggle"));
    expect(screen.queryByText("Test Org")).not.toBeInTheDocument();

    // Click expand.
    await user.click(screen.getByTestId("sidebar-toggle"));
    expect(screen.getByText("Test Org")).toBeInTheDocument();
  });

  it("passes unread count to topbar", () => {
    renderLayout({ unreadCount: 7 });
    expect(screen.getByTestId("notification-badge")).toHaveTextContent("7");
  });

  it("passes user menu to topbar", () => {
    renderLayout({ userMenu: <span>Avatar</span> });
    expect(screen.getByText("Avatar")).toBeInTheDocument();
  });

  it("passes onSearch to topbar", async () => {
    const user = userEvent.setup();
    const onSearch = vi.fn();
    renderLayout({ onSearch });

    const input = screen.getByTestId("search-input");
    await user.type(input, "query");
    await user.keyboard("{Enter}");

    expect(onSearch).toHaveBeenCalledWith("query");
  });
});
