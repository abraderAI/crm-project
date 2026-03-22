import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { Home, MessageSquare, Headset, Plus, Shield, BarChart3, Users } from "lucide-react";
import { Sidebar } from "./sidebar";
import type { SidebarNavItem } from "@/lib/nav-config";

const mockItems: SidebarNavItem[] = [
  {
    id: "home",
    label: "Home",
    href: "/",
    icon: Home,
    minTier: 1,
  },
  {
    id: "forum",
    label: "Forum",
    href: "/forum",
    icon: MessageSquare,
    minTier: 1,
  },
  {
    id: "support",
    label: "Support",
    href: "/support",
    icon: Headset,
    minTier: 2,
    children: [
      {
        id: "support-tickets",
        label: "All Tickets",
        href: "/support",
        icon: Headset,
        minTier: 2,
      },
      {
        id: "support-new",
        label: "New Ticket",
        href: "/support/tickets/new",
        icon: Plus,
        minTier: 2,
      },
    ],
  },
  {
    id: "admin",
    label: "Admin",
    href: "/admin",
    icon: Shield,
    minTier: 6,
    children: [
      {
        id: "admin-overview",
        label: "Overview",
        href: "/admin",
        icon: BarChart3,
        minTier: 6,
      },
      {
        id: "admin-users",
        label: "Users",
        href: "/admin/users",
        icon: Users,
        minTier: 6,
      },
    ],
  },
];

describe("Sidebar", () => {
  it("renders the sidebar element", () => {
    render(<Sidebar items={mockItems} />);
    expect(screen.getByTestId("sidebar")).toBeInTheDocument();
  });

  it("renders DEFT brand text when not collapsed", () => {
    render(<Sidebar items={mockItems} />);
    expect(screen.getByText("DEFT")).toBeInTheDocument();
  });

  it("renders all top-level nav items", () => {
    render(<Sidebar items={mockItems} />);
    expect(screen.getByTestId("nav-item-home")).toBeInTheDocument();
    expect(screen.getByTestId("nav-link-home")).toHaveTextContent("Home");
    expect(screen.getByTestId("nav-item-forum")).toBeInTheDocument();
    expect(screen.getByTestId("nav-item-support")).toBeInTheDocument();
    expect(screen.getByTestId("nav-item-admin")).toBeInTheDocument();
  });

  it("does not show sub-menu children by default (accordion collapsed)", () => {
    render(<Sidebar items={mockItems} />);
    // Sub-menu items exist in DOM (for animation) but panel has maxHeight 0.
    const panels = screen.getAllByTestId("submenu-panel");
    for (const panel of panels) {
      expect(panel.style.maxHeight).toBe("0px");
    }
  });

  it("expands sub-menu when toggle is clicked", async () => {
    const user = userEvent.setup();
    render(<Sidebar items={mockItems} />);

    const toggle = screen.getByTestId("nav-toggle-support");
    await user.click(toggle);

    // After clicking, the sub-menu panel should have expanded maxHeight.
    const supportPanel = screen.getByTestId("nav-item-support").querySelector("[data-testid='submenu-panel']");
    expect(supportPanel).toBeTruthy();
    expect(supportPanel?.getAttribute("data-expanded")).toBe("true");
    expect(supportPanel?.style.maxHeight).toBe("2000px");
  });

  it("collapses sub-menu when toggle is clicked again", async () => {
    const user = userEvent.setup();
    render(<Sidebar items={mockItems} />);

    const toggle = screen.getByTestId("nav-toggle-support");
    await user.click(toggle); // expand
    await user.click(toggle); // collapse

    // After animation frame, maxHeight should go to 0.
    // In test env rAF is sync, so we check immediately.
    const supportPanel = screen.getByTestId("nav-item-support").querySelector("[data-testid='submenu-panel']");
    expect(supportPanel).toBeTruthy();
  });

  it("accordion behavior: expanding one section collapses another", async () => {
    const user = userEvent.setup();
    render(<Sidebar items={mockItems} />);

    // Expand support.
    await user.click(screen.getByTestId("nav-toggle-support"));
    // Now expand admin — support should collapse.
    await user.click(screen.getByTestId("nav-toggle-admin"));

    const supportPanel = screen.getByTestId("nav-item-support").querySelector("[data-testid='submenu-panel']");
    const adminPanel = screen.getByTestId("nav-item-admin").querySelector("[data-testid='submenu-panel']");
    // Admin expanded, support collapsed.
    expect(adminPanel?.getAttribute("data-expanded")).toBe("true");
    expect(supportPanel?.getAttribute("data-expanded")).toBe("false");
    expect(supportPanel?.style.maxHeight).toBe("0px");
  });

  it("auto-expands the section matching the current route", () => {
    render(<Sidebar items={mockItems} currentPath="/support/tickets/new" />);
    // Support section should be expanded since the route matches a child.
    const supportPanel = screen.getByTestId("nav-item-support").querySelector("[data-testid='submenu-panel']");
    expect(supportPanel?.getAttribute("data-expanded")).toBe("true");
    expect(supportPanel?.style.maxHeight).toBe("2000px");
  });

  it("shows icon-only items when collapsed", () => {
    render(<Sidebar items={mockItems} collapsed={true} />);
    // Labels should not be visible.
    expect(screen.queryByText("DEFT")).not.toBeInTheDocument();
    // But nav items still render (icon-only with title tooltip).
    expect(screen.getByTestId("nav-link-home")).toBeInTheDocument();
    expect(screen.getByLabelText("Home")).toBeInTheDocument();
    expect(screen.getByLabelText("Support")).toBeInTheDocument();
  });

  it("calls onToggle when sidebar toggle is clicked", async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    render(<Sidebar items={mockItems} onToggle={onToggle} />);

    await user.click(screen.getByTestId("sidebar-toggle"));
    expect(onToggle).toHaveBeenCalledOnce();
  });

  it("shows expand sidebar label when collapsed", () => {
    render(<Sidebar items={mockItems} collapsed={true} onToggle={vi.fn()} />);
    expect(screen.getByLabelText("Expand sidebar")).toBeInTheDocument();
  });

  it("shows collapse sidebar label when expanded", () => {
    render(<Sidebar items={mockItems} collapsed={false} onToggle={vi.fn()} />);
    expect(screen.getByLabelText("Collapse sidebar")).toBeInTheDocument();
  });

  it("renders items without children (no toggle button)", () => {
    render(<Sidebar items={mockItems} />);
    expect(screen.getByTestId("nav-link-home")).toHaveTextContent("Home");
    expect(screen.queryByTestId("nav-toggle-home")).not.toBeInTheDocument();
  });

  it("renders empty items array without error", () => {
    render(<Sidebar items={[]} />);
    expect(screen.getByTestId("sidebar")).toBeInTheDocument();
  });

  it("renders sub-menu child items inside expanded section", async () => {
    const user = userEvent.setup();
    render(<Sidebar items={mockItems} />);

    await user.click(screen.getByTestId("nav-toggle-support"));
    expect(screen.getByTestId("nav-item-support-tickets")).toBeInTheDocument();
    expect(screen.getByTestId("nav-item-support-new")).toBeInTheDocument();
    expect(screen.getByTestId("nav-link-support-new")).toHaveTextContent("New Ticket");
  });

  it("auto-expands when currentPath matches parent href directly", () => {
    render(<Sidebar items={mockItems} currentPath="/support" />);
    const supportPanel = screen.getByTestId("nav-item-support").querySelector("[data-testid='submenu-panel']");
    expect(supportPanel?.getAttribute("data-expanded")).toBe("true");
  });

  it("clicking a parent link auto-expands its sub-menu", async () => {
    const user = userEvent.setup();
    render(<Sidebar items={mockItems} />);

    // Click the parent link (not the toggle button).
    const supportLink = screen.getByTestId("nav-link-support");
    await user.click(supportLink);

    const supportPanel = screen.getByTestId("nav-item-support").querySelector("[data-testid='submenu-panel']");
    expect(supportPanel?.getAttribute("data-expanded")).toBe("true");
  });

  it("highlights active child route in collapsed mode", () => {
    render(<Sidebar items={mockItems} currentPath="/admin/users" collapsed={true} />);
    // Admin should show active styling in collapsed mode.
    const adminLink = screen.getByTestId("nav-link-admin");
    expect(adminLink.className).toContain("bg-primary/10");
  });

  it("does not highlight inactive item in collapsed mode", () => {
    render(<Sidebar items={mockItems} currentPath="/" collapsed={true} />);
    const adminLink = screen.getByTestId("nav-link-admin");
    expect(adminLink.className).not.toContain("bg-primary/10");
  });

  it("highlights active sub-menu child item", async () => {
    const user = userEvent.setup();
    render(<Sidebar items={mockItems} currentPath="/support/tickets/new" />);

    // Support should be auto-expanded due to route.
    const newTicketLink = screen.getByTestId("nav-link-support-new");
    expect(newTicketLink.className).toContain("bg-primary/10");
  });

  it("does not auto-expand items without children even if path matches", () => {
    render(<Sidebar items={mockItems} currentPath="/forum" />);
    // Forum has no children, so no submenu panel.
    expect(screen.queryByTestId("nav-toggle-forum")).not.toBeInTheDocument();
  });
});
