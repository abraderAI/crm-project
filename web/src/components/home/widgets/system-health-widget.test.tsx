import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { SystemHealthWidget } from "./system-health-widget";

describe("SystemHealthWidget", () => {
  it("renders without crashing", () => {
    const { container } = render(<SystemHealthWidget token="tok" />);
    expect(container).toBeTruthy();
  });

  it("shows not-wired error after load", async () => {
    render(<SystemHealthWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("system-health-error")).toBeInTheDocument();
    });
    expect(screen.getByTestId("system-health-error")).toHaveTextContent(
      "Failed to load system health",
    );
  });
});
