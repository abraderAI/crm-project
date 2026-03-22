import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Tier6HomeScreen } from "./tier-6-home";
import { getDefaultLayout } from "@/lib/default-layouts";

describe("Tier6HomeScreen", () => {
  const defaultLayout = getDefaultLayout(6);

  it("renders the home screen container", () => {
    render(<Tier6HomeScreen token="tok" layout={defaultLayout} />);

    expect(screen.getByTestId("tier6-home-screen")).toBeInTheDocument();
    expect(screen.getByText("Platform Admin Dashboard")).toBeInTheDocument();
  });

  it("shows admin badge", () => {
    render(<Tier6HomeScreen token="tok" layout={defaultLayout} />);

    expect(screen.getByTestId("tier6-admin-badge")).toHaveTextContent("Admin");
  });

  it("renders admin quick links", () => {
    render(<Tier6HomeScreen token="tok" layout={defaultLayout} />);

    expect(screen.getByTestId("tier6-quick-links")).toBeInTheDocument();
    expect(screen.getByTestId("link-admin-dashboard")).toHaveAttribute("href", "/admin");
    expect(screen.getByTestId("link-support-report")).toHaveAttribute(
      "href",
      "/admin/reports/support",
    );
    expect(screen.getByTestId("link-sales-report")).toHaveAttribute("href", "/admin/reports/sales");
    expect(screen.getByTestId("link-user-management")).toHaveAttribute("href", "/admin/users");
    expect(screen.getByTestId("link-feature-flags")).toHaveAttribute(
      "href",
      "/admin/feature-flags",
    );
    expect(screen.getByTestId("link-audit-log")).toHaveAttribute("href", "/admin/audit-log");
  });

  it("links to admin console", () => {
    render(<Tier6HomeScreen token="tok" layout={defaultLayout} />);

    expect(screen.getByText("Admin Console")).toHaveAttribute("href", "/admin");
  });
});
