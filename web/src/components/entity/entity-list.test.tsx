import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { EntityCardProps } from "./entity-card";
import { EntityList } from "./entity-list";

const mockItems: EntityCardProps[] = [
  { id: "org-1", name: "Acme", slug: "acme", entityType: "org", href: "/orgs/acme" },
  { id: "org-2", name: "Beta", slug: "beta", entityType: "org", href: "/orgs/beta" },
];

describe("EntityList", () => {
  it("renders list container and title", () => {
    render(<EntityList items={mockItems} title="Organizations" />);
    expect(screen.getByTestId("entity-list")).toBeInTheDocument();
    expect(screen.getByText("Organizations")).toBeInTheDocument();
  });

  it("renders entity cards", () => {
    render(<EntityList items={mockItems} title="Organizations" />);
    expect(screen.getByTestId("entity-card-org-1")).toBeInTheDocument();
    expect(screen.getByTestId("entity-card-org-2")).toBeInTheDocument();
  });

  it("renders entity grid", () => {
    render(<EntityList items={mockItems} title="Organizations" />);
    expect(screen.getByTestId("entity-grid")).toBeInTheDocument();
  });

  it("shows empty state when no items", () => {
    render(<EntityList items={[]} title="Organizations" />);
    expect(screen.getByTestId("entity-list-empty")).toHaveTextContent("No items found.");
  });

  it("shows custom empty message", () => {
    render(<EntityList items={[]} title="Orgs" emptyMessage="Create your first org" />);
    expect(screen.getByTestId("entity-list-empty")).toHaveTextContent("Create your first org");
  });

  it("shows loading state when loading with no items", () => {
    render(<EntityList items={[]} title="Orgs" loading={true} />);
    expect(screen.getByTestId("entity-list-loading")).toHaveTextContent("Loading...");
  });

  it("does not show loading state when items exist", () => {
    render(<EntityList items={mockItems} title="Orgs" loading={true} />);
    expect(screen.queryByTestId("entity-list-loading")).not.toBeInTheDocument();
    expect(screen.getByTestId("entity-grid")).toBeInTheDocument();
  });

  it("shows create button when showCreate is true", () => {
    render(<EntityList items={mockItems} title="Orgs" showCreate={true} />);
    expect(screen.getByTestId("entity-create-btn")).toBeInTheDocument();
    expect(screen.getByTestId("entity-create-btn")).toHaveTextContent("Create");
  });

  it("does not show create button by default", () => {
    render(<EntityList items={mockItems} title="Orgs" />);
    expect(screen.queryByTestId("entity-create-btn")).not.toBeInTheDocument();
  });

  it("uses custom create label", () => {
    render(<EntityList items={mockItems} title="Orgs" showCreate={true} createLabel="New Org" />);
    expect(screen.getByTestId("entity-create-btn")).toHaveTextContent("New Org");
  });

  it("calls onCreate when create button clicked", async () => {
    const user = userEvent.setup();
    const onCreate = vi.fn();
    render(<EntityList items={mockItems} title="Orgs" showCreate={true} onCreate={onCreate} />);
    await user.click(screen.getByTestId("entity-create-btn"));
    expect(onCreate).toHaveBeenCalledOnce();
  });

  it("shows load more button when hasMore is true", () => {
    render(<EntityList items={mockItems} title="Orgs" hasMore={true} />);
    expect(screen.getByTestId("entity-load-more")).toHaveTextContent("Load more");
  });

  it("does not show load more when hasMore is false", () => {
    render(<EntityList items={mockItems} title="Orgs" hasMore={false} />);
    expect(screen.queryByTestId("entity-load-more")).not.toBeInTheDocument();
  });

  it("calls onLoadMore when load more clicked", async () => {
    const user = userEvent.setup();
    const onLoadMore = vi.fn();
    render(<EntityList items={mockItems} title="Orgs" hasMore={true} onLoadMore={onLoadMore} />);
    await user.click(screen.getByTestId("entity-load-more"));
    expect(onLoadMore).toHaveBeenCalledOnce();
  });

  it("shows loading text on load more button when loading", () => {
    render(<EntityList items={mockItems} title="Orgs" hasMore={true} loading={true} />);
    expect(screen.getByTestId("entity-load-more")).toHaveTextContent("Loading...");
  });

  it("disables load more button when loading", () => {
    render(<EntityList items={mockItems} title="Orgs" hasMore={true} loading={true} />);
    expect(screen.getByTestId("entity-load-more")).toBeDisabled();
  });
});
