import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { Revision } from "@/lib/api-types";
import { RevisionHistory } from "./revision-history";

const mockRevisions: Revision[] = [
  {
    id: "r-1",
    entity_type: "message",
    entity_id: "m-1",
    version: 2,
    previous_content: "old content",
    editor_id: "user-1",
    created_at: "2025-01-15T12:00:00Z",
    updated_at: "2025-01-15T12:00:00Z",
  },
  {
    id: "r-2",
    entity_type: "message",
    entity_id: "m-1",
    version: 1,
    previous_content: "original content",
    editor_id: "user-2",
    created_at: "2025-01-15T10:00:00Z",
    updated_at: "2025-01-15T10:00:00Z",
  },
];

describe("RevisionHistory", () => {
  it("renders empty state when no revisions", () => {
    render(<RevisionHistory revisions={[]} />);
    expect(screen.getByTestId("revision-history-empty")).toHaveTextContent("No revision history.");
  });

  it("renders revision list", () => {
    render(<RevisionHistory revisions={mockRevisions} />);
    expect(screen.getByTestId("revision-history")).toBeInTheDocument();
    expect(screen.getByTestId("revision-item-r-1")).toBeInTheDocument();
    expect(screen.getByTestId("revision-item-r-2")).toBeInTheDocument();
  });

  it("shows revision count", () => {
    render(<RevisionHistory revisions={mockRevisions} />);
    expect(screen.getByText("Revision history (2)")).toBeInTheDocument();
  });

  it("shows version number", () => {
    render(<RevisionHistory revisions={mockRevisions} />);
    expect(screen.getByTestId("revision-version-r-1")).toHaveTextContent("v2");
    expect(screen.getByTestId("revision-version-r-2")).toHaveTextContent("v1");
  });

  it("shows editor ID", () => {
    render(<RevisionHistory revisions={mockRevisions} />);
    expect(screen.getByTestId("revision-editor-r-1")).toHaveTextContent("user-1");
    expect(screen.getByTestId("revision-editor-r-2")).toHaveTextContent("user-2");
  });

  it("calls onViewRevision when revision clicked", async () => {
    const user = userEvent.setup();
    const onView = vi.fn();
    render(<RevisionHistory revisions={mockRevisions} onViewRevision={onView} />);

    await user.click(screen.getByTestId("revision-item-r-1"));
    expect(onView).toHaveBeenCalledWith(mockRevisions[0]);
  });

  it("highlights selected revision", () => {
    render(<RevisionHistory revisions={mockRevisions} selectedId="r-1" />);
    const item = screen.getByTestId("revision-item-r-1");
    expect(item.className).toContain("bg-accent");
  });

  it("does not highlight non-selected revisions", () => {
    render(<RevisionHistory revisions={mockRevisions} selectedId="r-1" />);
    const item = screen.getByTestId("revision-item-r-2");
    expect(item.className).not.toContain("bg-accent ");
  });
});
