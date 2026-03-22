import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { TicketStatsWidget } from "./ticket-stats-widget";

describe("TicketStatsWidget", () => {
  it("renders without crashing", () => {
    const { container } = render(<TicketStatsWidget token="tok" />);
    expect(container).toBeTruthy();
  });

  it("shows not-wired error after load", async () => {
    render(<TicketStatsWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("ticket-stats-error")).toBeInTheDocument();
    });
    expect(screen.getByTestId("ticket-stats-error")).toHaveTextContent(
      "Failed to load ticket stats",
    );
  });
});
