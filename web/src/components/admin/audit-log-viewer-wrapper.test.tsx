import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: vi.fn().mockResolvedValue(null) }),
}));

import { AuditLogViewerWithDirectory } from "./audit-log-viewer-wrapper";

describe("AuditLogViewerWithDirectory", () => {
  it("renders AuditLogViewer with entries", () => {
    render(<AuditLogViewerWithDirectory entries={[]} />);
    expect(screen.getByTestId("audit-log-viewer")).toBeInTheDocument();
  });

  it("passes entries through to AuditLogViewer", () => {
    const entries = [
      {
        id: "a1",
        user_id: "user-1",
        action: "create",
        entity_type: "org",
        entity_id: "org-123",
        created_at: "2026-03-21T00:00:00Z",
        updated_at: "2026-03-21T00:00:00Z",
      },
    ];
    render(<AuditLogViewerWithDirectory entries={entries} />);
    expect(screen.getByTestId("audit-item-a1")).toBeInTheDocument();
  });
});
