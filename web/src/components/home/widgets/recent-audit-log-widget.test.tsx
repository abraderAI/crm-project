import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RecentAuditLogWidget } from "./recent-audit-log-widget";

describe("RecentAuditLogWidget", () => {
  it("renders and shows not-wired error", async () => {
    render(<RecentAuditLogWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("audit-log-error")).toBeInTheDocument();
    });
    expect(screen.getByTestId("audit-log-error")).toHaveTextContent("Failed to load audit events");
  });
});
