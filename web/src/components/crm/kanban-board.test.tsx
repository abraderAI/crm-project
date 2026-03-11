import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { Thread } from "@/lib/api-types";
import { KanbanBoard } from "./kanban-board";

function makeThread(overrides: Partial<Thread> = {}): Thread {
  return {
    id: "t-1",
    board_id: "b-1",
    title: "Lead: Acme",
    slug: "lead-acme",
    metadata: '{"company":"Acme","value":50000,"assigned_to":"alice","score":75}',
    author_id: "u-1",
    is_pinned: false,
    is_locked: false,
    is_hidden: false,
    vote_score: 0,
    stage: "new_lead",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
    ...overrides,
  };
}

describe("KanbanBoard", () => {
  it("renders board container", () => {
    render(<KanbanBoard threads={[]} />);
    expect(screen.getByTestId("kanban-board")).toBeInTheDocument();
  });

  it("renders all 8 stage columns by default", () => {
    render(<KanbanBoard threads={[]} />);
    expect(screen.getByTestId("kanban-columns")).toBeInTheDocument();
    expect(screen.getByTestId("kanban-column-new_lead")).toBeInTheDocument();
    expect(screen.getByTestId("kanban-column-contacted")).toBeInTheDocument();
    expect(screen.getByTestId("kanban-column-qualified")).toBeInTheDocument();
    expect(screen.getByTestId("kanban-column-proposal")).toBeInTheDocument();
    expect(screen.getByTestId("kanban-column-negotiation")).toBeInTheDocument();
    expect(screen.getByTestId("kanban-column-closed_won")).toBeInTheDocument();
    expect(screen.getByTestId("kanban-column-closed_lost")).toBeInTheDocument();
    expect(screen.getByTestId("kanban-column-nurturing")).toBeInTheDocument();
  });

  it("renders only specified stages", () => {
    render(<KanbanBoard threads={[]} stages={["new_lead", "qualified"]} />);
    expect(screen.getByTestId("kanban-column-new_lead")).toBeInTheDocument();
    expect(screen.getByTestId("kanban-column-qualified")).toBeInTheDocument();
    expect(screen.queryByTestId("kanban-column-contacted")).not.toBeInTheDocument();
  });

  it("renders column headers with stage labels", () => {
    render(<KanbanBoard threads={[]} stages={["new_lead"]} />);
    expect(screen.getByTestId("kanban-header-new_lead")).toHaveTextContent("New Lead");
  });

  it("shows loading state", () => {
    render(<KanbanBoard threads={[]} loading />);
    expect(screen.getByTestId("kanban-loading")).toHaveTextContent("Loading pipeline...");
    expect(screen.queryByTestId("kanban-board")).not.toBeInTheDocument();
  });

  it("places leads in correct columns", () => {
    const threads = [
      makeThread({ id: "t-1", stage: "new_lead" }),
      makeThread({ id: "t-2", stage: "qualified" }),
    ];
    render(<KanbanBoard threads={threads} stages={["new_lead", "qualified"]} />);
    expect(screen.getByTestId("kanban-card-t-1")).toBeInTheDocument();
    expect(screen.getByTestId("kanban-card-t-2")).toBeInTheDocument();
  });

  it("shows empty message for columns with no leads", () => {
    render(<KanbanBoard threads={[]} stages={["new_lead"]} />);
    expect(screen.getByTestId("kanban-empty-new_lead")).toHaveTextContent("No leads");
  });

  it("shows stage count in badge", () => {
    const threads = [
      makeThread({ id: "t-1", stage: "new_lead" }),
      makeThread({ id: "t-2", stage: "new_lead" }),
    ];
    render(<KanbanBoard threads={threads} stages={["new_lead"]} />);
    expect(screen.getByTestId("stage-badge-new_lead")).toHaveTextContent("2");
  });

  it("renders filters", () => {
    render(<KanbanBoard threads={[]} />);
    expect(screen.getByTestId("kanban-filters")).toBeInTheDocument();
  });

  it("filters by search text", async () => {
    const user = userEvent.setup();
    const threads = [
      makeThread({ id: "t-1", title: "Lead: Acme" }),
      makeThread({ id: "t-2", title: "Lead: Beta Corp", metadata: '{"company":"Beta"}' }),
    ];
    render(<KanbanBoard threads={threads} stages={["new_lead"]} />);

    // Both visible initially
    expect(screen.getByTestId("kanban-card-t-1")).toBeInTheDocument();
    expect(screen.getByTestId("kanban-card-t-2")).toBeInTheDocument();

    // Type search
    await user.type(screen.getByTestId("kanban-search-input"), "Beta");
    expect(screen.queryByTestId("kanban-card-t-1")).not.toBeInTheDocument();
    expect(screen.getByTestId("kanban-card-t-2")).toBeInTheDocument();
  });

  it("filters by assignee", async () => {
    const user = userEvent.setup();
    const threads = [
      makeThread({ id: "t-1", metadata: '{"assigned_to":"alice"}' }),
      makeThread({ id: "t-2", metadata: '{"assigned_to":"bob"}' }),
    ];
    render(<KanbanBoard threads={threads} stages={["new_lead"]} />);

    await user.selectOptions(screen.getByTestId("kanban-assignee-filter"), "alice");
    expect(screen.getByTestId("kanban-card-t-1")).toBeInTheDocument();
    expect(screen.queryByTestId("kanban-card-t-2")).not.toBeInTheDocument();
  });

  it("calls onCardClick when card is clicked", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(<KanbanBoard threads={[makeThread()]} onCardClick={onClick} stages={["new_lead"]} />);
    await user.click(screen.getByTestId("kanban-card-t-1"));
    expect(onClick).toHaveBeenCalledWith("t-1");
  });

  it("generates card links from basePath", () => {
    render(<KanbanBoard threads={[makeThread()]} basePath="/crm/leads" stages={["new_lead"]} />);
    const link = screen.getByTestId("kanban-card-link-t-1");
    expect(link).toHaveAttribute("href", "/crm/leads/lead-acme");
  });

  it("handles drag over on column", () => {
    render(<KanbanBoard threads={[]} stages={["new_lead"]} />);
    const column = screen.getByTestId("kanban-column-new_lead");
    fireEvent.dragOver(column, {
      dataTransfer: { dropEffect: "" },
    });
    // Column should get highlighted styling
    expect(column.className).toContain("border-primary");
  });

  it("handles drag leave on column", () => {
    render(<KanbanBoard threads={[]} stages={["new_lead"]} />);
    const column = screen.getByTestId("kanban-column-new_lead");
    fireEvent.dragOver(column, { dataTransfer: { dropEffect: "" } });
    fireEvent.dragLeave(column);
    expect(column.className).not.toContain("border-primary");
  });

  it("calls onStageChange on drop", () => {
    const onStageChange = vi.fn();
    render(
      <KanbanBoard
        threads={[makeThread()]}
        onStageChange={onStageChange}
        stages={["new_lead", "qualified"]}
      />,
    );
    const qualifiedColumn = screen.getByTestId("kanban-column-qualified");
    fireEvent.drop(qualifiedColumn, {
      dataTransfer: { getData: () => "t-1" },
    });
    expect(onStageChange).toHaveBeenCalledWith("t-1", "qualified");
  });

  it("does not call onStageChange when not provided", () => {
    render(<KanbanBoard threads={[makeThread()]} stages={["new_lead", "qualified"]} />);
    const qualifiedColumn = screen.getByTestId("kanban-column-qualified");
    // Should not throw
    fireEvent.drop(qualifiedColumn, {
      dataTransfer: { getData: () => "t-1" },
    });
  });

  it("does not call onStageChange with empty thread ID", () => {
    const onStageChange = vi.fn();
    render(<KanbanBoard threads={[]} onStageChange={onStageChange} stages={["new_lead"]} />);
    const column = screen.getByTestId("kanban-column-new_lead");
    fireEvent.drop(column, {
      dataTransfer: { getData: () => "" },
    });
    expect(onStageChange).not.toHaveBeenCalled();
  });

  it("populates assignee filter options from lead data", () => {
    const threads = [
      makeThread({ id: "t-1", metadata: '{"assigned_to":"bob"}' }),
      makeThread({ id: "t-2", metadata: '{"assigned_to":"alice"}' }),
    ];
    render(<KanbanBoard threads={threads} />);
    const select = screen.getByTestId("kanban-assignee-filter");
    expect(select).toContainHTML("alice");
    expect(select).toContainHTML("bob");
  });
});
