import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { Sidebar, type NavItem } from "./sidebar";

const mockItems: NavItem[] = [
  {
    id: "org-1",
    label: "Acme Corp",
    href: "/orgs/acme",
    type: "org",
    children: [
      {
        id: "space-1",
        label: "Sales",
        href: "/orgs/acme/spaces/sales",
        type: "space",
        children: [
          {
            id: "board-1",
            label: "Pipeline",
            href: "/orgs/acme/spaces/sales/boards/pipeline",
            type: "board",
          },
        ],
      },
      {
        id: "space-2",
        label: "Support",
        href: "/orgs/acme/spaces/support",
        type: "space",
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
    expect(screen.getByTestId("nav-item-org-1")).toBeInTheDocument();
    expect(screen.getByTestId("nav-link-org-1")).toHaveTextContent("Acme Corp");
  });

  it("renders nested children by default (expanded)", () => {
    render(<Sidebar items={mockItems} />);
    expect(screen.getByTestId("nav-item-space-1")).toBeInTheDocument();
    expect(screen.getByTestId("nav-item-board-1")).toBeInTheDocument();
  });

  it("collapses children when toggle is clicked", async () => {
    const user = userEvent.setup();
    render(<Sidebar items={mockItems} />);

    const toggle = screen.getByTestId("nav-toggle-org-1");
    await user.click(toggle);

    expect(screen.queryByTestId("nav-item-space-1")).not.toBeInTheDocument();
  });

  it("re-expands children after collapse", async () => {
    const user = userEvent.setup();
    render(<Sidebar items={mockItems} />);

    const toggle = screen.getByTestId("nav-toggle-org-1");
    await user.click(toggle); // collapse
    await user.click(toggle); // expand
    expect(screen.getByTestId("nav-item-space-1")).toBeInTheDocument();
  });

  it("hides navigation when collapsed", () => {
    render(<Sidebar items={mockItems} collapsed={true} />);
    expect(screen.queryByText("Acme Corp")).not.toBeInTheDocument();
    expect(screen.queryByText("DEFT")).not.toBeInTheDocument();
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

  it("renders items without children (leaf nodes)", () => {
    render(<Sidebar items={mockItems} />);
    expect(screen.getByTestId("nav-link-space-2")).toHaveTextContent("Support");
    // Space-2 has no children, so no toggle button.
    expect(screen.queryByTestId("nav-toggle-space-2")).not.toBeInTheDocument();
  });

  it("renders empty items array without error", () => {
    render(<Sidebar items={[]} />);
    expect(screen.getByTestId("sidebar")).toBeInTheDocument();
  });
});
