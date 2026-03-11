import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { EntityCard } from "./entity-card";

describe("EntityCard", () => {
  const baseProps = {
    id: "org-1",
    name: "Acme Corp",
    slug: "acme-corp",
    entityType: "org" as const,
  };

  it("renders entity name and slug", () => {
    render(<EntityCard {...baseProps} />);
    expect(screen.getByText("Acme Corp")).toBeInTheDocument();
    expect(screen.getByText(/acme-corp/)).toBeInTheDocument();
  });

  it("renders correct label for org type", () => {
    render(<EntityCard {...baseProps} />);
    expect(screen.getByText(/Organization/)).toBeInTheDocument();
  });

  it("renders correct label for space type", () => {
    render(<EntityCard {...baseProps} entityType="space" />);
    expect(screen.getByText(/Space/)).toBeInTheDocument();
  });

  it("renders correct label for board type", () => {
    render(<EntityCard {...baseProps} entityType="board" />);
    expect(screen.getByText(/Board/)).toBeInTheDocument();
  });

  it("renders icon", () => {
    render(<EntityCard {...baseProps} />);
    expect(screen.getByTestId("entity-icon")).toBeInTheDocument();
  });

  it("renders description when provided", () => {
    render(<EntityCard {...baseProps} description="A test organization" />);
    expect(screen.getByTestId("entity-description")).toHaveTextContent("A test organization");
  });

  it("does not render description when not provided", () => {
    render(<EntityCard {...baseProps} />);
    expect(screen.queryByTestId("entity-description")).not.toBeInTheDocument();
  });

  it("renders metadata tags from JSON", () => {
    const metadata = JSON.stringify({ tier: "premium", region: "us-west" });
    render(<EntityCard {...baseProps} metadata={metadata} />);
    const metadataEl = screen.getByTestId("entity-metadata");
    expect(metadataEl).toBeInTheDocument();
    expect(screen.getByText("tier: premium")).toBeInTheDocument();
    expect(screen.getByText("region: us-west")).toBeInTheDocument();
  });

  it("limits metadata display to 3 items", () => {
    const metadata = JSON.stringify({ a: 1, b: 2, c: 3, d: 4 });
    render(<EntityCard {...baseProps} metadata={metadata} />);
    const metadataEl = screen.getByTestId("entity-metadata");
    expect(metadataEl.children).toHaveLength(3);
  });

  it("handles empty metadata gracefully", () => {
    render(<EntityCard {...baseProps} metadata="{}" />);
    expect(screen.queryByTestId("entity-metadata")).not.toBeInTheDocument();
  });

  it("handles invalid metadata JSON gracefully", () => {
    render(<EntityCard {...baseProps} metadata="invalid-json" />);
    expect(screen.queryByTestId("entity-metadata")).not.toBeInTheDocument();
  });

  it("handles array metadata gracefully", () => {
    render(<EntityCard {...baseProps} metadata="[1,2,3]" />);
    expect(screen.queryByTestId("entity-metadata")).not.toBeInTheDocument();
  });

  it("calls onClick when clicked", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(<EntityCard {...baseProps} onClick={onClick} />);
    await user.click(screen.getByTestId("entity-card-org-1"));
    expect(onClick).toHaveBeenCalledWith("org-1");
  });

  it("calls onClick on Enter key", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(<EntityCard {...baseProps} onClick={onClick} />);
    const card = screen.getByTestId("entity-card-org-1");
    card.focus();
    await user.keyboard("{Enter}");
    expect(onClick).toHaveBeenCalledWith("org-1");
  });

  it("calls onClick on Space key", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(<EntityCard {...baseProps} onClick={onClick} />);
    const card = screen.getByTestId("entity-card-org-1");
    card.focus();
    await user.keyboard(" ");
    expect(onClick).toHaveBeenCalledWith("org-1");
  });

  it("has role button when onClick provided", () => {
    render(<EntityCard {...baseProps} onClick={vi.fn()} />);
    expect(screen.getByTestId("entity-card-org-1")).toHaveAttribute("role", "button");
  });

  it("does not have role button when no onClick", () => {
    render(<EntityCard {...baseProps} />);
    expect(screen.getByTestId("entity-card-org-1")).not.toHaveAttribute("role");
  });

  it("renders as link when href provided", () => {
    render(<EntityCard {...baseProps} href="/orgs/acme" />);
    const link = screen.getByTestId("entity-link-org-1");
    expect(link.tagName).toBe("A");
    expect(link).toHaveAttribute("href", "/orgs/acme");
  });

  it("renders without link wrapper when no href", () => {
    render(<EntityCard {...baseProps} />);
    expect(screen.queryByTestId("entity-link-org-1")).not.toBeInTheDocument();
  });

  it("renders with empty string metadata", () => {
    render(<EntityCard {...baseProps} metadata="" />);
    expect(screen.queryByTestId("entity-metadata")).not.toBeInTheDocument();
  });
});
