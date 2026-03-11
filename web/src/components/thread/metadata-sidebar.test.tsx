import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { MetadataSidebar } from "./metadata-sidebar";

describe("MetadataSidebar", () => {
  it("renders sidebar container", () => {
    render(<MetadataSidebar voteScore={0} />);
    expect(screen.getByTestId("metadata-sidebar")).toBeInTheDocument();
    expect(screen.getByText("Details")).toBeInTheDocument();
  });

  it("shows vote score", () => {
    render(<MetadataSidebar voteScore={42} />);
    expect(screen.getByTestId("sidebar-votes")).toHaveTextContent("42");
  });

  it("shows status when provided", () => {
    render(<MetadataSidebar voteScore={0} status="open" />);
    expect(screen.getByTestId("sidebar-status")).toHaveTextContent("open");
  });

  it("shows priority when provided", () => {
    render(<MetadataSidebar voteScore={0} priority="high" />);
    expect(screen.getByTestId("sidebar-priority")).toHaveTextContent("high");
  });

  it("shows stage when provided", () => {
    render(<MetadataSidebar voteScore={0} stage="new_lead" />);
    expect(screen.getByTestId("sidebar-stage")).toHaveTextContent("new_lead");
  });

  it("shows assigned_to when provided", () => {
    render(<MetadataSidebar voteScore={0} assignedTo="user-123" />);
    expect(screen.getByTestId("sidebar-assigned")).toHaveTextContent("user-123");
  });

  it("does not show status when not provided", () => {
    render(<MetadataSidebar voteScore={0} />);
    expect(screen.queryByTestId("sidebar-status")).not.toBeInTheDocument();
  });

  it("shows pinned flag", () => {
    render(<MetadataSidebar voteScore={0} isPinned={true} />);
    expect(screen.getByTestId("sidebar-pinned")).toHaveTextContent("Yes");
  });

  it("shows locked flag", () => {
    render(<MetadataSidebar voteScore={0} isLocked={true} />);
    expect(screen.getByTestId("sidebar-locked")).toHaveTextContent("Yes");
  });

  it("does not show flags section when neither pinned nor locked", () => {
    render(<MetadataSidebar voteScore={0} />);
    expect(screen.queryByTestId("sidebar-pinned")).not.toBeInTheDocument();
    expect(screen.queryByTestId("sidebar-locked")).not.toBeInTheDocument();
  });

  it("shows custom metadata fields from JSON string", () => {
    render(<MetadataSidebar voteScore={0} metadata='{"company":"Acme","deal_size":"50000"}' />);
    expect(screen.getByTestId("sidebar-custom-company")).toHaveTextContent("Acme");
    expect(screen.getByTestId("sidebar-custom-deal_size")).toHaveTextContent("50000");
  });

  it("shows custom metadata fields from object", () => {
    render(<MetadataSidebar voteScore={0} metadata={{ source: "web", region: "NA" }} />);
    expect(screen.getByTestId("sidebar-custom-source")).toHaveTextContent("web");
    expect(screen.getByTestId("sidebar-custom-region")).toHaveTextContent("NA");
  });

  it("filters out reserved metadata keys", () => {
    render(
      <MetadataSidebar
        voteScore={0}
        status="open"
        metadata={{ status: "open", custom_field: "value" }}
      />,
    );
    // status shown via dedicated field, not in custom
    expect(screen.queryByTestId("sidebar-custom-status")).not.toBeInTheDocument();
    expect(screen.getByTestId("sidebar-custom-custom_field")).toHaveTextContent("value");
  });

  it("handles undefined metadata gracefully", () => {
    render(<MetadataSidebar voteScore={0} />);
    expect(screen.getByTestId("metadata-sidebar")).toBeInTheDocument();
  });
});
