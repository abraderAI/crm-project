import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ThemeProvider } from "@/components/theme-provider";
import { Topbar } from "./topbar";

function renderTopbar(props: Parameters<typeof Topbar>[0] = {}): ReturnType<typeof render> {
  return render(
    <ThemeProvider>
      <Topbar {...props} />
    </ThemeProvider>,
  );
}

describe("Topbar", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("renders the topbar element", () => {
    renderTopbar();
    expect(screen.getByTestId("topbar")).toBeInTheDocument();
  });

  it("renders search input", () => {
    renderTopbar();
    expect(screen.getByTestId("search-input")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Search...")).toBeInTheDocument();
  });

  it("renders notification bell", () => {
    renderTopbar();
    expect(screen.getByTestId("notification-bell")).toBeInTheDocument();
  });

  it("does not show badge when unreadCount is 0", () => {
    renderTopbar({ unreadCount: 0 });
    expect(screen.queryByTestId("notification-badge")).not.toBeInTheDocument();
  });

  it("shows badge with count when unreadCount > 0", () => {
    renderTopbar({ unreadCount: 5 });
    const badge = screen.getByTestId("notification-badge");
    expect(badge).toBeInTheDocument();
    expect(badge.textContent).toBe("5");
  });

  it("caps badge at 99+", () => {
    renderTopbar({ unreadCount: 150 });
    const badge = screen.getByTestId("notification-badge");
    expect(badge.textContent).toBe("99+");
  });

  it("notification bell has correct aria-label with count", () => {
    renderTopbar({ unreadCount: 3 });
    expect(screen.getByLabelText("3 unread notifications")).toBeInTheDocument();
  });

  it("notification bell has correct aria-label with zero count", () => {
    renderTopbar({ unreadCount: 0 });
    expect(screen.getByLabelText("No unread notifications")).toBeInTheDocument();
  });

  it("renders theme toggle", () => {
    renderTopbar();
    expect(screen.getByTestId("theme-toggle")).toBeInTheDocument();
  });

  it("renders user menu when provided", () => {
    renderTopbar({ userMenu: <span>User</span> });
    expect(screen.getByTestId("user-menu")).toBeInTheDocument();
    expect(screen.getByText("User")).toBeInTheDocument();
  });

  it("does not render user menu when not provided", () => {
    renderTopbar();
    expect(screen.queryByTestId("user-menu")).not.toBeInTheDocument();
  });

  it("calls onSearch when form is submitted", async () => {
    const user = userEvent.setup();
    const onSearch = vi.fn();
    renderTopbar({ onSearch });

    const input = screen.getByTestId("search-input");
    await user.type(input, "test query");
    await user.keyboard("{Enter}");

    expect(onSearch).toHaveBeenCalledWith("test query");
  });

  it("updates search input value on typing", async () => {
    const user = userEvent.setup();
    renderTopbar();

    const input = screen.getByTestId("search-input") as HTMLInputElement;
    await user.type(input, "hello");
    expect(input.value).toBe("hello");
  });
});
