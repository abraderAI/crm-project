import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RecentLeadsWidget } from "./recent-leads-widget";

describe("RecentLeadsWidget", () => {
  it("renders and shows not-wired error", async () => {
    render(<RecentLeadsWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("recent-leads-error")).toBeInTheDocument();
    });
    expect(screen.getByTestId("recent-leads-error")).toHaveTextContent("Failed to load recent leads");
  });
});
