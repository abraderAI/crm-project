import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LeadPipelineWidget } from "./lead-pipeline-widget";

describe("LeadPipelineWidget", () => {
  it("renders without crashing", () => {
    const { container } = render(<LeadPipelineWidget token="tok" />);
    expect(container).toBeTruthy();
  });

  it("shows not-wired error after load", async () => {
    render(<LeadPipelineWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("lead-pipeline-error")).toBeInTheDocument();
    });
    expect(screen.getByTestId("lead-pipeline-error")).toHaveTextContent(
      "Failed to load lead pipeline data",
    );
  });
});
