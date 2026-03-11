import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { EntityList, type EntityListItem } from "./entity-list";

const items: EntityListItem[] = [
  { id: "1", name: "Acme Corp", slug: "acme-corp", description: "First org" },
  { id: "2", name: "Beta Inc", slug: "beta-inc" },
  { id: "3", name: "Gamma LLC", slug: "gamma-llc", metadata: '{"tier":"pro"}' },
];

describe("EntityList", () => {
  it("renders the list title", () => {
    render(<EntityList entityType="org" items={items} title="Organizations" />);
    expect(screen.getByText("Organizations")).toBeInTheDocument();
  });

  it("renders all entity cards", () => {
    render(<EntityList entityType="org" items={items} title="Organizations" />);
    expect(screen.getByTestId("entity-card-1")).toBeInTheDocument();
    expect(screen.getByTestId("entity-card-2")).toBeInTheDocument();
    expect(screen.getByTestId("entity-card-3")).toBeInTheDocument();
  });

  it("renders entity grid container", () => {
    render(<EntityList entityType="org" items={items} title="Organizations" />);
    expect(screen.getByTestId("entity-grid")).toBeInTheDocument();
  });

  it("shows empty state when no items", () => {
    render(<EntityList entityType="org" items={[]} title="Organizations" />);
    expect(screen.getByTestId("empty-state")).toHaveTextContent("No organizations found.");
  });

  it("shows loading state", () => {
    render(<EntityList entityType="org" items={[]} title="Organizations" loading={true} />);
    expect(screen.getByTestId("loading-state")).toHaveTextContent("Loading...");
    expect(screen.queryByTestId("empty-state")).not.toBeInTheDocument();
  });

  it("hides entity grid when loading", () => {
    render(<EntityList entityType="org" items={items} title="Organizations" loading={true} />);
    expect(screen.queryByTestId("entity-grid")).not.toBeInTheDocument();
  });

  it("renders create button when onCreate provided", () => {
    render(<EntityList entityType="org" items={items} title="Organizations" onCreate={vi.fn()} />);
    expect(screen.getByTestId("entity-create-btn")).toBeInTheDocument();
  });

  it("does not render create button when onCreate not provided", () => {
    render(<EntityList entityType="org" items={items} title="Organizations" />);
    expect(screen.queryByTestId("entity-create-btn")).not.toBeInTheDocument();
  });

  it("calls onCreate when create button clicked", async () => {
    const user = userEvent.setup();
    const onCreate = vi.fn();
    render(<EntityList entityType="org" items={items} title="Organizations" onCreate={onCreate} />);

    await user.click(screen.getByTestId("entity-create-btn"));
    expect(onCreate).toHaveBeenCalledOnce();
  });

  it("calls onSelect when entity card clicked", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(<EntityList entityType="org" items={items} title="Organizations" onSelect={onSelect} />);

    await user.click(screen.getByTestId("entity-card-1"));
    expect(onSelect).toHaveBeenCalledWith("1");
  });

  it("renders load more button when hasMore is true", () => {
    render(
      <EntityList
        entityType="org"
        items={items}
        title="Organizations"
        hasMore={true}
        onLoadMore={vi.fn()}
      />,
    );
    expect(screen.getByTestId("load-more-btn")).toBeInTheDocument();
  });

  it("hides load more when hasMore is false", () => {
    render(<EntityList entityType="org" items={items} title="Organizations" hasMore={false} />);
    expect(screen.queryByTestId("load-more-btn")).not.toBeInTheDocument();
  });

  it("calls onLoadMore when load more clicked", async () => {
    const user = userEvent.setup();
    const onLoadMore = vi.fn();
    render(
      <EntityList
        entityType="org"
        items={items}
        title="Organizations"
        hasMore={true}
        onLoadMore={onLoadMore}
      />,
    );

    await user.click(screen.getByTestId("load-more-btn"));
    expect(onLoadMore).toHaveBeenCalledOnce();
  });

  it("hides load more when loading", () => {
    render(
      <EntityList
        entityType="org"
        items={items}
        title="Organizations"
        hasMore={true}
        loading={true}
      />,
    );
    expect(screen.queryByTestId("load-more-btn")).not.toBeInTheDocument();
  });

  it("passes getHref to cards", () => {
    const getHref = (item: EntityListItem): string => `/orgs/${item.slug}`;
    render(<EntityList entityType="org" items={items} title="Organizations" getHref={getHref} />);
    expect(screen.getByTestId("entity-link-1")).toHaveAttribute("href", "/orgs/acme-corp");
  });
});
