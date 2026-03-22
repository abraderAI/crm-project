import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { TicketStatsWidget } from "./ticket-stats-widget";

describe("TicketStatsWidget", () => {
  it("renders and shows not-wired error", async () => {
    render(<TicketStatsWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("ticket-stats-error")).toBeInTheDocument();
    });
    expect(screen.getByTestId("ticket-stats-error")).toHaveTextContent("Failed to load ticket stats");
  });
});
