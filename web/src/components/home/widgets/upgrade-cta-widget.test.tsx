import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { UpgradeCTAWidget } from "./upgrade-cta-widget";

vi.mock("next/link", () => ({
  default: ({
    children,
    href,
    ...rest
  }: {
    children: React.ReactNode;
    href: string;
    className?: string;
    "data-testid"?: string;
  }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}));

describe("UpgradeCTAWidget", () => {
  it("renders the widget container", () => {
    render(<UpgradeCTAWidget />);
    expect(screen.getByTestId("upgrade-cta-widget")).toBeInTheDocument();
  });

  it("displays upgrade to pro heading", () => {
    render(<UpgradeCTAWidget />);
    expect(screen.getByText("Upgrade to Pro")).toBeInTheDocument();
  });

  it("displays description text", () => {
    render(<UpgradeCTAWidget />);
    expect(
      screen.getByText("Unlock the full power of the DEFT platform for your team."),
    ).toBeInTheDocument();
  });

  it("lists pro benefits", () => {
    render(<UpgradeCTAWidget />);
    const benefits = screen.getByTestId("upgrade-benefits");
    expect(benefits.querySelectorAll("li")).toHaveLength(3);
    expect(screen.getByText("Organization workspace with team collaboration")).toBeInTheDocument();
    expect(screen.getByText("Priority support with dedicated SLAs")).toBeInTheDocument();
    expect(screen.getByText("Advanced reporting and analytics")).toBeInTheDocument();
  });

  it("renders upgrade link pointing to /upgrade", () => {
    render(<UpgradeCTAWidget />);
    const link = screen.getByTestId("upgrade-cta-link");
    expect(link).toHaveAttribute("href", "/upgrade");
    expect(link).toHaveTextContent("Upgrade now");
  });
});
