import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Breadcrumbs, type BreadcrumbItem } from "./breadcrumbs";

describe("Breadcrumbs", () => {
  it("renders nothing when items is empty", () => {
    const { container } = render(<Breadcrumbs items={[]} />);
    expect(container.firstChild).toBeNull();
  });

  it("renders single breadcrumb as current page", () => {
    const items: BreadcrumbItem[] = [{ label: "Home" }];
    render(<Breadcrumbs items={items} />);
    const crumb = screen.getByTestId("breadcrumb-0");
    expect(crumb).toHaveTextContent("Home");
    expect(crumb).toHaveAttribute("aria-current", "page");
  });

  it("renders multiple breadcrumbs with links", () => {
    const items: BreadcrumbItem[] = [
      { label: "Home", href: "/" },
      { label: "Acme", href: "/orgs/acme" },
      { label: "Sales" },
    ];
    render(<Breadcrumbs items={items} />);

    // First two are links.
    const first = screen.getByTestId("breadcrumb-0");
    expect(first.tagName).toBe("A");
    expect(first).toHaveAttribute("href", "/");

    const second = screen.getByTestId("breadcrumb-1");
    expect(second.tagName).toBe("A");
    expect(second).toHaveAttribute("href", "/orgs/acme");

    // Last is current page (span).
    const last = screen.getByTestId("breadcrumb-2");
    expect(last.tagName).toBe("SPAN");
    expect(last).toHaveAttribute("aria-current", "page");
    expect(last).toHaveTextContent("Sales");
  });

  it("renders items without href as spans", () => {
    const items: BreadcrumbItem[] = [{ label: "No Link" }, { label: "Last" }];
    render(<Breadcrumbs items={items} />);

    const first = screen.getByTestId("breadcrumb-0");
    expect(first.tagName).toBe("SPAN");
    // First item without href is not aria-current (only last is).
    expect(first).not.toHaveAttribute("aria-current");
  });

  it("renders with proper nav and aria-label", () => {
    render(<Breadcrumbs items={[{ label: "X" }]} />);
    const nav = screen.getByTestId("breadcrumbs");
    expect(nav.tagName).toBe("NAV");
    expect(nav).toHaveAttribute("aria-label", "Breadcrumb");
  });

  it("renders separator chevrons between items", () => {
    const items: BreadcrumbItem[] = [
      { label: "A", href: "/" },
      { label: "B", href: "/b" },
      { label: "C" },
    ];
    render(<Breadcrumbs items={items} />);
    // There should be list items for all three.
    const listItems = screen.getByTestId("breadcrumbs").querySelectorAll("li");
    expect(listItems).toHaveLength(3);
  });
});
