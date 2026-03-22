import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { TicketQueueWidget } from "./ticket-queue-widget";

describe("TicketQueueWidget", () => {
  it("renders without crashing", () => {
    const { container } = render(<TicketQueueWidget token="tok" />);
    expect(container).toBeTruthy();
  });

  it("shows not-wired error after load", async () => {
    render(<TicketQueueWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("ticket-queue-error")).toBeInTheDocument();
    });
    expect(screen.getByTestId("ticket-queue-error")).toHaveTextContent(
      "Failed to load ticket queue",
    );
  });
});
