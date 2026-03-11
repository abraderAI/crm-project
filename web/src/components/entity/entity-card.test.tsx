import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { EntityCard, metadataKeyCount, parseMetadata } from "./entity-card";

describe("parseMetadata", () => {
  it("returns empty object for undefined", () => {
    expect(parseMetadata(undefined)).toEqual({});
  });

  it("returns empty object for empty string", () => {
    expect(parseMetadata("")).toEqual({});
  });

  it("parses valid JSON string", () => {
    expect(parseMetadata('{"key":"value"}')).toEqual({ key: "value" });
  });

  it("returns object directly if already an object", () => {
    const obj = { key: "value" };
    expect(parseMetadata(obj)).toBe(obj);
  });

  it("returns empty object for invalid JSON", () => {
    expect(parseMetadata("not json")).toEqual({});
  });

  it("returns empty object for JSON array", () => {
    expect(parseMetadata("[1,2,3]")).toEqual({});
  });

  it("returns empty object for JSON primitive", () => {
    expect(parseMetadata("42")).toEqual({});
  });
});

describe("metadataKeyCount", () => {
  it("returns 0 for undefined", () => {
    expect(metadataKeyCount(undefined)).toBe(0);
  });

  it("returns 0 for empty object", () => {
    expect(metadataKeyCount({})).toBe(0);
  });

  it("returns correct count", () => {
    expect(metadataKeyCount({ a: 1, b: 2, c: 3 })).toBe(3);
  });
});

describe("EntityCard", () => {
  it("renders card with org type", () => {
    render(
      <EntityCard id="org-1" name="Acme Corp" slug="acme" entityType="org" href="/orgs/acme" />,
    );
    expect(screen.getByTestId("entity-card-org-1")).toBeInTheDocument();
    expect(screen.getByText("Acme Corp")).toBeInTheDocument();
    expect(screen.getByText("Organization")).toBeInTheDocument();
    expect(screen.getByTestId("entity-card-slug-org-1")).toHaveTextContent("/acme");
  });

  it("renders card with space type and spaceType label", () => {
    render(
      <EntityCard
        id="space-1"
        name="Sales"
        slug="sales"
        entityType="space"
        spaceType="crm"
        href="/orgs/acme/spaces/sales"
      />,
    );
    expect(screen.getByText("Sales")).toBeInTheDocument();
    expect(screen.getByText("crm")).toBeInTheDocument();
  });

  it("renders card with board type", () => {
    render(
      <EntityCard
        id="board-1"
        name="Pipeline"
        slug="pipeline"
        entityType="board"
        href="/orgs/acme/spaces/sales/boards/pipeline"
      />,
    );
    expect(screen.getByText("Pipeline")).toBeInTheDocument();
    expect(screen.getByText("Board")).toBeInTheDocument();
  });

  it("renders description when provided", () => {
    render(
      <EntityCard
        id="org-1"
        name="Acme"
        slug="acme"
        entityType="org"
        href="/orgs/acme"
        description="A great organization"
      />,
    );
    expect(screen.getByTestId("entity-card-desc-org-1")).toHaveTextContent("A great organization");
  });

  it("does not render description when absent", () => {
    render(<EntityCard id="org-1" name="Acme" slug="acme" entityType="org" href="/orgs/acme" />);
    expect(screen.queryByTestId("entity-card-desc-org-1")).not.toBeInTheDocument();
  });

  it("shows metadata field count when metadata has keys", () => {
    render(
      <EntityCard
        id="org-1"
        name="Acme"
        slug="acme"
        entityType="org"
        href="/orgs/acme"
        metadata={{ tier: "pro", status: "active" }}
      />,
    );
    expect(screen.getByTestId("entity-card-meta-org-1")).toHaveTextContent("2 metadata fields");
  });

  it("shows singular metadata field label for single key", () => {
    render(
      <EntityCard
        id="org-1"
        name="Acme"
        slug="acme"
        entityType="org"
        href="/orgs/acme"
        metadata={{ tier: "pro" }}
      />,
    );
    expect(screen.getByTestId("entity-card-meta-org-1")).toHaveTextContent("1 metadata field");
  });

  it("does not show metadata count when metadata is empty", () => {
    render(
      <EntityCard
        id="org-1"
        name="Acme"
        slug="acme"
        entityType="org"
        href="/orgs/acme"
        metadata={{}}
      />,
    );
    expect(screen.queryByTestId("entity-card-meta-org-1")).not.toBeInTheDocument();
  });

  it("links to the correct href", () => {
    render(<EntityCard id="org-1" name="Acme" slug="acme" entityType="org" href="/orgs/acme" />);
    const link = screen.getByTestId("entity-card-org-1");
    expect(link.tagName).toBe("A");
    expect(link).toHaveAttribute("href", "/orgs/acme");
  });
});
