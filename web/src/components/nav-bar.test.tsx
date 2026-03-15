import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

// Mock next/navigation.
let mockPathname = "/";
vi.mock("next/navigation", () => ({
  usePathname: () => mockPathname,
}));

// Mock @clerk/nextjs.
vi.mock("@clerk/nextjs", () => ({
  UserButton: () => <div data-testid="clerk-user-button" />,
  useAuth: () => ({ getToken: vi.fn().mockResolvedValue(null) }),
}));

// Mock NavNotificationBell to isolate NavBar tests.
vi.mock("./nav-notification-bell", () => ({
  NavNotificationBell: () => <div data-testid="nav-notification-bell" />,
}));

// Mock ThemeToggle to isolate NavBar tests.
vi.mock("./theme-toggle", () => ({
  ThemeToggle: () => <div data-testid="theme-toggle" />,
}));

import { NavBar } from "./nav-bar";

describe("NavBar", () => {
  it("renders the logo", () => {
    mockPathname = "/";
    render(<NavBar />);
    expect(screen.getByTestId("nav-logo")).toHaveTextContent("DEFT Evolution");
  });

  it("renders all nav links", () => {
    mockPathname = "/";
    render(<NavBar />);
    expect(screen.getByTestId("nav-link-home")).toBeInTheDocument();
    expect(screen.getByTestId("nav-link-crm")).toBeInTheDocument();
    expect(screen.getByTestId("nav-link-reports")).toBeInTheDocument();
    expect(screen.getByTestId("nav-link-search")).toBeInTheDocument();
    expect(screen.getByTestId("nav-link-admin")).toBeInTheDocument();
  });

  it("highlights Home link when on /", () => {
    mockPathname = "/";
    render(<NavBar />);
    expect(screen.getByTestId("nav-link-home")).toHaveClass("bg-accent");
    expect(screen.getByTestId("nav-link-admin")).not.toHaveClass("bg-accent");
  });

  it("highlights Admin link when on /admin", () => {
    mockPathname = "/admin";
    render(<NavBar />);
    expect(screen.getByTestId("nav-link-admin")).toHaveClass("bg-accent");
    expect(screen.getByTestId("nav-link-home")).not.toHaveClass("bg-accent");
  });

  it("highlights Admin link on admin sub-pages", () => {
    mockPathname = "/admin/users";
    render(<NavBar />);
    expect(screen.getByTestId("nav-link-admin")).toHaveClass("bg-accent");
  });

  it("highlights CRM link when on /crm", () => {
    mockPathname = "/crm";
    render(<NavBar />);
    expect(screen.getByTestId("nav-link-crm")).toHaveClass("bg-accent");
    expect(screen.getByTestId("nav-link-home")).not.toHaveClass("bg-accent");
  });

  it("renders NavNotificationBell component", () => {
    mockPathname = "/";
    render(<NavBar />);
    expect(screen.getByTestId("nav-notification-bell")).toBeInTheDocument();
  });

  it("highlights Search link when on /search", () => {
    mockPathname = "/search";
    render(<NavBar />);
    expect(screen.getByTestId("nav-link-search")).toHaveClass("bg-accent");
  });

  it("renders the Clerk UserButton", () => {
    mockPathname = "/";
    render(<NavBar />);
    expect(screen.getByTestId("nav-user-button")).toBeInTheDocument();
    expect(screen.getByTestId("clerk-user-button")).toBeInTheDocument();
  });

  it("renders the nav-bar container", () => {
    mockPathname = "/";
    render(<NavBar />);
    expect(screen.getByTestId("nav-bar")).toBeInTheDocument();
  });

  it("renders the ThemeToggle", () => {
    mockPathname = "/";
    render(<NavBar />);
    expect(screen.getByTestId("theme-toggle")).toBeInTheDocument();
  });
});
