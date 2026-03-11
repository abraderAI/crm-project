import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { RevisionHistory, type Revision } from "./revision-history";

const mockRevisions: Revision[] = [
  {
    id: "rev-3",
    version: 3,
    editorId: "user-1",
    previousContent: "Version 2 content",
    createdAt: "2025-01-15T12:00:00Z",
  },
  {
    id: "rev-2",
    version: 2,
    editorId: "user-2",
    previousContent: "Version 1 content",
    createdAt: "2025-01-15T11:00:00Z",
  },
  {
    id: "rev-1",
    version: 1,
    editorId: "user-1",
    previousContent: "Original content",
    createdAt: "2025-01-15T10:00:00Z",
  },
];

describe("RevisionHistory", () => {
  it("renders no revisions message when empty", () => {
    render(<RevisionHistory revisions={[]} />);
    expect(screen.getByTestId("no-revisions")).toHaveTextContent("No revision history.");
  });

  it("does not render revision list when empty", () => {
    render(<RevisionHistory revisions={[]} />);
    expect(screen.queryByTestId("revision-history")).not.toBeInTheDocument();
  });

  it("renders revision history container", () => {
    render(<RevisionHistory revisions={mockRevisions} />);
    expect(screen.getByTestId("revision-history")).toBeInTheDocument();
  });

  it("shows revision count in header", () => {
    render(<RevisionHistory revisions={mockRevisions} />);
    expect(screen.getByText(/Revision History \(3\)/)).toBeInTheDocument();
  });

  it("renders all revisions", () => {
    render(<RevisionHistory revisions={mockRevisions} />);
    expect(screen.getByTestId("revision-rev-3")).toBeInTheDocument();
    expect(screen.getByTestId("revision-rev-2")).toBeInTheDocument();
    expect(screen.getByTestId("revision-rev-1")).toBeInTheDocument();
  });

  it("renders version numbers", () => {
    render(<RevisionHistory revisions={mockRevisions} />);
    expect(screen.getByText("v3")).toBeInTheDocument();
    expect(screen.getByText("v2")).toBeInTheDocument();
    expect(screen.getByText("v1")).toBeInTheDocument();
  });

  it("renders editor IDs", () => {
    render(<RevisionHistory revisions={mockRevisions} />);
    expect(screen.getAllByText(/by user-1/)).toHaveLength(2);
    expect(screen.getByText(/by user-2/)).toBeInTheDocument();
  });

  it("does not show content preview initially", () => {
    render(<RevisionHistory revisions={mockRevisions} />);
    expect(screen.queryByTestId("revision-content")).not.toBeInTheDocument();
  });

  it("shows content preview when revision clicked", async () => {
    const user = userEvent.setup();
    render(<RevisionHistory revisions={mockRevisions} />);

    await user.click(screen.getByTestId("revision-rev-2"));
    expect(screen.getByTestId("revision-content")).toBeInTheDocument();
    expect(screen.getByText("Version 1 content")).toBeInTheDocument();
  });

  it("hides content preview when same revision clicked again", async () => {
    const user = userEvent.setup();
    render(<RevisionHistory revisions={mockRevisions} />);

    await user.click(screen.getByTestId("revision-rev-2"));
    expect(screen.getByTestId("revision-content")).toBeInTheDocument();

    await user.click(screen.getByTestId("revision-rev-2"));
    expect(screen.queryByTestId("revision-content")).not.toBeInTheDocument();
  });

  it("calls onSelect when revision clicked", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(<RevisionHistory revisions={mockRevisions} onSelect={onSelect} />);

    await user.click(screen.getByTestId("revision-rev-3"));
    expect(onSelect).toHaveBeenCalledWith(mockRevisions[0]);
  });

  it("switches content preview between revisions", async () => {
    const user = userEvent.setup();
    render(<RevisionHistory revisions={mockRevisions} />);

    await user.click(screen.getByTestId("revision-rev-3"));
    expect(screen.getByText("Version 2 content")).toBeInTheDocument();

    await user.click(screen.getByTestId("revision-rev-1"));
    expect(screen.getByText("Original content")).toBeInTheDocument();
  });

  it("renders revision list container", () => {
    render(<RevisionHistory revisions={mockRevisions} />);
    expect(screen.getByTestId("revision-list")).toBeInTheDocument();
  });
});
