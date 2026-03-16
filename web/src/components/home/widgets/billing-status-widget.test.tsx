import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { BillingStatusWidget } from "./billing-status-widget";

describe("BillingStatusWidget", () => {
  it("renders billing info for org owners", () => {
    render(<BillingStatusWidget isOwner />);
    expect(screen.getByTestId("billing-status-widget")).toBeInTheDocument();
    expect(screen.getByTestId("billing-plan")).toHaveTextContent("Pro Plan");
    expect(screen.getByTestId("billing-renewal")).toBeInTheDocument();
  });

  it("renders manage billing link for owners", () => {
    render(<BillingStatusWidget isOwner />);
    const link = screen.getByTestId("billing-manage-link");
    expect(link).toHaveAttribute("href", "/admin/billing");
    expect(link).toHaveTextContent("Manage billing");
  });

  it("renders nothing for non-owners", () => {
    const { container } = render(<BillingStatusWidget isOwner={false} />);
    expect(container.innerHTML).toBe("");
  });

  it("shows placeholder renewal date", () => {
    render(<BillingStatusWidget isOwner />);
    expect(screen.getByTestId("billing-renewal")).toHaveTextContent("Renews: April 15, 2026");
  });
});
