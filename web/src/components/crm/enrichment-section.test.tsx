import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { EnrichmentData } from "@/lib/crm-types";
import { EnrichmentSection } from "./enrichment-section";

const fullEnrichment: EnrichmentData = {
  summary: "Acme Corp is a promising enterprise lead with strong buying signals.",
  next_action: "Schedule a demo call with the CTO.",
  enriched_at: "2025-06-15T10:00:00Z",
};

describe("EnrichmentSection", () => {
  it("renders container", () => {
    render(<EnrichmentSection enrichment={null} />);
    expect(screen.getByTestId("enrichment-section")).toBeInTheDocument();
  });

  it("shows empty state when no enrichment", () => {
    render(<EnrichmentSection enrichment={null} />);
    expect(screen.getByTestId("enrichment-empty")).toHaveTextContent("No AI enrichment data yet");
  });

  it("renders summary when present", () => {
    render(<EnrichmentSection enrichment={fullEnrichment} />);
    expect(screen.getByTestId("enrichment-summary")).toBeInTheDocument();
    expect(screen.getByText(/Acme Corp is a promising/)).toBeInTheDocument();
  });

  it("renders next action when present", () => {
    render(<EnrichmentSection enrichment={fullEnrichment} />);
    expect(screen.getByTestId("enrichment-next-action")).toBeInTheDocument();
    expect(screen.getByText(/Schedule a demo call/)).toBeInTheDocument();
  });

  it("renders enriched timestamp", () => {
    render(<EnrichmentSection enrichment={fullEnrichment} />);
    expect(screen.getByTestId("enrichment-timestamp")).toBeInTheDocument();
  });

  it("hides summary when not present", () => {
    render(<EnrichmentSection enrichment={{ enriched_at: "2025-01-01" }} />);
    expect(screen.queryByTestId("enrichment-summary")).not.toBeInTheDocument();
  });

  it("hides next action when not present", () => {
    render(<EnrichmentSection enrichment={{ summary: "test" }} />);
    expect(screen.queryByTestId("enrichment-next-action")).not.toBeInTheDocument();
  });

  it("hides timestamp when not present", () => {
    render(<EnrichmentSection enrichment={{ summary: "test" }} />);
    expect(screen.queryByTestId("enrichment-timestamp")).not.toBeInTheDocument();
  });

  it("renders Enrich button when callback provided", () => {
    render(<EnrichmentSection enrichment={null} onEnrich={vi.fn()} />);
    expect(screen.getByTestId("enrich-button")).toBeInTheDocument();
    expect(screen.getByTestId("enrich-button")).toHaveTextContent("Enrich");
  });

  it("hides Enrich button when no callback", () => {
    render(<EnrichmentSection enrichment={null} />);
    expect(screen.queryByTestId("enrich-button")).not.toBeInTheDocument();
  });

  it("calls onEnrich when clicked", async () => {
    const user = userEvent.setup();
    const onEnrich = vi.fn();
    render(<EnrichmentSection enrichment={null} onEnrich={onEnrich} />);
    await user.click(screen.getByTestId("enrich-button"));
    expect(onEnrich).toHaveBeenCalledOnce();
  });

  it("shows loading state on button", () => {
    render(<EnrichmentSection enrichment={null} onEnrich={vi.fn()} loading />);
    expect(screen.getByTestId("enrich-button")).toHaveTextContent("Enriching...");
    expect(screen.getByTestId("enrich-button")).toBeDisabled();
  });

  it("shows enrichment content when data provided", () => {
    render(<EnrichmentSection enrichment={fullEnrichment} />);
    expect(screen.getByTestId("enrichment-content")).toBeInTheDocument();
    expect(screen.queryByTestId("enrichment-empty")).not.toBeInTheDocument();
  });
});
