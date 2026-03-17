import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { ThemeProvider } from "@/components/theme-provider";

// Mock next/navigation.
const mockPush = vi.fn();
vi.mock("next/navigation", () => ({
  usePathname: () => "/crm",
  useRouter: () => ({ push: mockPush }),
}));

// Mock Clerk auth.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
  UserButton: () => <div data-testid="clerk-user-button">User</div>,
}));

// Mock useNotifications hook.
vi.mock("@/hooks/use-notifications", () => ({
  useNotifications: () => ({
    notifications: [],
    unreadCount: 3,
    loading: false,
    error: null,
    markRead: vi.fn(),
    markAllRead: vi.fn(),
    handleWSNotification: vi.fn(),
    refresh: vi.fn(),
  }),
}));

// Mock ChatbotWidget loaded via next/dynamic.
vi.mock("@/components/chatbot-widget", () => ({
  ChatbotWidget: () => <div data-testid="chatbot-widget">ChatbotWidget</div>,
}));

import { AppLayoutWrapper } from "./app-layout-wrapper";

/** Render helper wrapping component in ThemeProvider. */
function renderWithTheme(ui: React.ReactNode): ReturnType<typeof render> {
  return render(<ThemeProvider>{ui}</ThemeProvider>);
}

describe("AppLayoutWrapper", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
  });

  it("renders AppLayout shell", () => {
    renderWithTheme(<AppLayoutWrapper>Content</AppLayoutWrapper>);
    expect(screen.getByTestId("app-layout")).toBeInTheDocument();
  });

  it("renders sidebar with nav items", () => {
    renderWithTheme(<AppLayoutWrapper>Content</AppLayoutWrapper>);
    expect(screen.getByTestId("sidebar")).toBeInTheDocument();
    expect(screen.getByTestId("nav-link-home")).toBeInTheDocument();
    expect(screen.getByTestId("nav-link-forum")).toBeInTheDocument();
    expect(screen.getByTestId("nav-link-docs")).toBeInTheDocument();
    expect(screen.getByTestId("nav-link-support")).toBeInTheDocument();
    expect(screen.getByTestId("nav-link-notifications")).toBeInTheDocument();
    expect(screen.getByTestId("nav-link-crm")).toBeInTheDocument();
    expect(screen.getByTestId("nav-link-search")).toBeInTheDocument();
    expect(screen.getByTestId("nav-link-admin")).toBeInTheDocument();
  });

  it("renders topbar", () => {
    renderWithTheme(<AppLayoutWrapper>Content</AppLayoutWrapper>);
    expect(screen.getByTestId("topbar")).toBeInTheDocument();
  });

  it("shows unread notification count in topbar", () => {
    renderWithTheme(<AppLayoutWrapper>Content</AppLayoutWrapper>);
    expect(screen.getByTestId("notification-badge")).toHaveTextContent("3");
  });

  it("renders children in the content area", () => {
    renderWithTheme(
      <AppLayoutWrapper>
        <div data-testid="child">Hello</div>
      </AppLayoutWrapper>,
    );
    expect(screen.getByTestId("child")).toBeInTheDocument();
  });

  it("renders Clerk UserButton in user menu", () => {
    renderWithTheme(<AppLayoutWrapper>Content</AppLayoutWrapper>);
    expect(screen.getByTestId("clerk-user-button")).toBeInTheDocument();
  });

  it("navigates to search page on search submit", async () => {
    const user = userEvent.setup();
    renderWithTheme(<AppLayoutWrapper>Content</AppLayoutWrapper>);

    const input = screen.getByTestId("search-input");
    await user.type(input, "test query");
    await user.keyboard("{Enter}");

    expect(mockPush).toHaveBeenCalledWith("/search?q=test+query");
  });
});
