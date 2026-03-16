import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { LlmUsageEntry } from "@/lib/api-types";
import { LlmUsageTable } from "./llm-usage-table";

const fixtureEntries: LlmUsageEntry[] = [
  {
    id: "llm-1",
    endpoint: "/v1/chat/message",
    model: "gpt-4o",
    input_tokens: 1200,
    output_tokens: 450,
    duration_ms: 3200,
    created_at: "2026-03-15T10:30:00Z",
  },
  {
    id: "llm-2",
    endpoint: "/v1/threads/enrich",
    model: "gpt-4o-mini",
    input_tokens: 800,
    output_tokens: 200,
    duration_ms: 1500,
    created_at: "2026-03-15T09:15:00Z",
  },
  {
    id: "llm-3",
    endpoint: "/v1/chat/message",
    model: "gpt-4o",
    input_tokens: 2500,
    output_tokens: 1100,
    duration_ms: 5800,
    created_at: "2026-03-14T22:45:00Z",
  },
];

describe("LlmUsageTable", () => {
  it("renders the heading", () => {
    render(<LlmUsageTable entries={[]} />);
    expect(screen.getByText("LLM Usage Log")).toBeInTheDocument();
  });

  it("shows empty state when no entries", () => {
    render(<LlmUsageTable entries={[]} />);
    expect(screen.getByTestId("llm-usage-empty")).toBeInTheDocument();
  });

  it("shows loading state", () => {
    render(<LlmUsageTable entries={[]} loading={true} />);
    expect(screen.getByTestId("llm-usage-loading")).toBeInTheDocument();
  });

  it("renders table with entries", () => {
    render(<LlmUsageTable entries={fixtureEntries} />);
    expect(screen.getByTestId("llm-usage-table")).toBeInTheDocument();
  });

  it("renders column headers", () => {
    render(<LlmUsageTable entries={fixtureEntries} />);
    expect(screen.getByText("Endpoint")).toBeInTheDocument();
    expect(screen.getByText("Model")).toBeInTheDocument();
    expect(screen.getByText("Input Tokens")).toBeInTheDocument();
    expect(screen.getByText("Output Tokens")).toBeInTheDocument();
    expect(screen.getByText("Latency")).toBeInTheDocument();
    expect(screen.getByText("Timestamp")).toBeInTheDocument();
  });

  it("renders all rows", () => {
    render(<LlmUsageTable entries={fixtureEntries} />);
    const rows = screen.getAllByTestId(/^llm-usage-row-/);
    expect(rows).toHaveLength(3);
  });

  it("displays model name for each row", () => {
    render(<LlmUsageTable entries={fixtureEntries} />);
    expect(screen.getByTestId("llm-usage-model-llm-1")).toHaveTextContent("gpt-4o");
    expect(screen.getByTestId("llm-usage-model-llm-2")).toHaveTextContent("gpt-4o-mini");
  });

  it("displays input tokens for each row", () => {
    render(<LlmUsageTable entries={fixtureEntries} />);
    expect(screen.getByTestId("llm-usage-input-llm-1")).toHaveTextContent("1,200");
    expect(screen.getByTestId("llm-usage-input-llm-2")).toHaveTextContent("800");
  });

  it("displays output tokens for each row", () => {
    render(<LlmUsageTable entries={fixtureEntries} />);
    expect(screen.getByTestId("llm-usage-output-llm-1")).toHaveTextContent("450");
    expect(screen.getByTestId("llm-usage-output-llm-2")).toHaveTextContent("200");
  });

  it("displays latency in ms", () => {
    render(<LlmUsageTable entries={fixtureEntries} />);
    expect(screen.getByTestId("llm-usage-latency-llm-1")).toHaveTextContent("3,200 ms");
    expect(screen.getByTestId("llm-usage-latency-llm-3")).toHaveTextContent("5,800 ms");
  });

  it("displays endpoint for each row", () => {
    render(<LlmUsageTable entries={fixtureEntries} />);
    expect(screen.getByTestId("llm-usage-endpoint-llm-1")).toHaveTextContent("/v1/chat/message");
    expect(screen.getByTestId("llm-usage-endpoint-llm-2")).toHaveTextContent("/v1/threads/enrich");
  });

  it("displays formatted timestamp", () => {
    render(<LlmUsageTable entries={fixtureEntries} />);
    expect(screen.getByTestId("llm-usage-time-llm-1")).toBeInTheDocument();
    // Verify timestamp cell is not empty.
    expect(screen.getByTestId("llm-usage-time-llm-1").textContent).not.toBe("");
  });
});
