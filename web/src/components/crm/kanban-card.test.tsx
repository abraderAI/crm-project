import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { LeadCard } from "@/lib/crm-types";
import type { Thread } from "@/lib/api-types";
import { KanbanCard, StageBadge } from "./kanban-card";

function makeCard(
  overrides: Partial<Thread> = {},
  leadOverrides: Partial<LeadCard["lead"]> = {},
): LeadCard {
  return {
    thread: {
      id: "t-1",
      board_id: "b-1",
      title: "Lead: Acme Corp",
      slug: "lead-acme-corp",
      metadata: "{}",
      author_id: "u-1",
      is_pinned: false,
      is_locked: false,
      is_hidden: false,
      vote_score: 0,
      stage: "new_lead",
      created_at: "2025-01-01T00:00:00Z",
      updated_at: "2025-01-01T00:00:00Z",
      ...overrides,
    },
    lead: {
      company: "Acme Corp",
      value: 50000,
      assigned_to: "alice",
      score: 75,
      ...leadOverrides,
    },
    stage: "new_lead",
  };
}

describe("KanbanCard", () => {
  it("renders card with title", () => {
    render(<KanbanCard card={makeCard()} />);
    expect(screen.getByTestId("kanban-card-t-1")).toBeInTheDocument();
    expect(screen.getByTestId("kanban-card-title-t-1")).toHaveTextContent("Lead: Acme Corp");
  });

  it("renders company name", () => {
    render(<KanbanCard card={makeCard()} />);
    expect(screen.getByTestId("kanban-card-company-t-1")).toHaveTextContent("Acme Corp");
  });

  it("hides company when not present", () => {
    render(<KanbanCard card={makeCard({}, { company: undefined })} />);
    expect(screen.queryByTestId("kanban-card-company-t-1")).not.toBeInTheDocument();
  });

  it("renders deal value", () => {
    render(<KanbanCard card={makeCard()} />);
    expect(screen.getByTestId("kanban-card-value-t-1")).toHaveTextContent("$50,000");
  });

  it("hides value when zero", () => {
    render(<KanbanCard card={makeCard({}, { value: 0 })} />);
    expect(screen.queryByTestId("kanban-card-value-t-1")).not.toBeInTheDocument();
  });

  it("hides value when undefined", () => {
    render(<KanbanCard card={makeCard({}, { value: undefined })} />);
    expect(screen.queryByTestId("kanban-card-value-t-1")).not.toBeInTheDocument();
  });

  it("renders score", () => {
    render(<KanbanCard card={makeCard()} />);
    expect(screen.getByTestId("kanban-card-score-t-1")).toHaveTextContent("75");
  });

  it("hides score when not present", () => {
    render(<KanbanCard card={makeCard({}, { score: undefined })} />);
    expect(screen.queryByTestId("kanban-card-score-t-1")).not.toBeInTheDocument();
  });

  it("renders assignee", () => {
    render(<KanbanCard card={makeCard()} />);
    expect(screen.getByTestId("kanban-card-assignee-t-1")).toHaveTextContent("alice");
  });

  it("hides assignee when not present", () => {
    render(<KanbanCard card={makeCard({}, { assigned_to: undefined })} />);
    expect(screen.queryByTestId("kanban-card-assignee-t-1")).not.toBeInTheDocument();
  });

  it("calls onClick when clicked", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(<KanbanCard card={makeCard()} onClick={onClick} />);
    await user.click(screen.getByTestId("kanban-card-t-1"));
    expect(onClick).toHaveBeenCalledWith("t-1");
  });

  it("calls onClick on Enter key", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(<KanbanCard card={makeCard()} onClick={onClick} />);
    const card = screen.getByTestId("kanban-card-t-1");
    card.focus();
    await user.keyboard("{Enter}");
    expect(onClick).toHaveBeenCalledWith("t-1");
  });

  it("calls onClick on Space key", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(<KanbanCard card={makeCard()} onClick={onClick} />);
    const card = screen.getByTestId("kanban-card-t-1");
    card.focus();
    await user.keyboard(" ");
    expect(onClick).toHaveBeenCalledWith("t-1");
  });

  it("renders as link when href provided", () => {
    render(<KanbanCard card={makeCard()} href="/leads/acme" />);
    const link = screen.getByTestId("kanban-card-link-t-1");
    expect(link.tagName).toBe("A");
    expect(link).toHaveAttribute("href", "/leads/acme");
  });

  it("does not render link wrapper without href", () => {
    render(<KanbanCard card={makeCard()} />);
    expect(screen.queryByTestId("kanban-card-link-t-1")).not.toBeInTheDocument();
  });

  it("is draggable", () => {
    render(<KanbanCard card={makeCard()} />);
    expect(screen.getByTestId("kanban-card-t-1")).toHaveAttribute("draggable", "true");
  });

  it("sets drag data on drag start", () => {
    render(<KanbanCard card={makeCard()} />);
    const card = screen.getByTestId("kanban-card-t-1");
    const setData = vi.fn();
    fireEvent.dragStart(card, {
      dataTransfer: { setData, effectAllowed: "" },
    });
    expect(setData).toHaveBeenCalledWith("text/plain", "t-1");
  });

  it("applies dragging styles when isDragging", () => {
    render(<KanbanCard card={makeCard()} isDragging />);
    const card = screen.getByTestId("kanban-card-t-1");
    expect(card.className).toContain("opacity-50");
  });

  it("does not apply dragging styles by default", () => {
    render(<KanbanCard card={makeCard()} />);
    const card = screen.getByTestId("kanban-card-t-1");
    expect(card.className).not.toContain("opacity-50");
  });

  it("sets button role when onClick is provided", () => {
    render(<KanbanCard card={makeCard()} onClick={vi.fn()} />);
    expect(screen.getByTestId("kanban-card-t-1")).toHaveAttribute("role", "button");
  });

  it("does not set button role without onClick", () => {
    render(<KanbanCard card={makeCard()} />);
    expect(screen.getByTestId("kanban-card-t-1")).not.toHaveAttribute("role");
  });

  it("renders deal_amount when present", () => {
    render(<KanbanCard card={makeCard({}, { deal_amount: 100000, value: 50000 })} />);
    expect(screen.getByTestId("kanban-card-deal-t-1")).toHaveTextContent("$100,000");
    // value should not render when deal_amount is present
    expect(screen.queryByTestId("kanban-card-value-t-1")).not.toBeInTheDocument();
  });

  it("renders weighted_forecast when present", () => {
    render(<KanbanCard card={makeCard({}, { weighted_forecast: 50000 })} />);
    expect(screen.getByTestId("kanban-card-forecast-t-1")).toHaveTextContent("$50,000");
  });

  it("renders expected close date", () => {
    render(<KanbanCard card={makeCard({}, { expected_close_date: "2099-12-31" })} />);
    expect(screen.getByTestId("kanban-card-close-date-t-1")).toBeInTheDocument();
  });

  it("renders overdue badge for past close dates", () => {
    render(<KanbanCard card={makeCard({}, { expected_close_date: "2020-01-01" })} />);
    expect(screen.getByTestId("kanban-card-overdue-t-1")).toHaveTextContent("Overdue");
  });

  it("does not render overdue badge for future close dates", () => {
    render(<KanbanCard card={makeCard({}, { expected_close_date: "2099-12-31" })} />);
    expect(screen.queryByTestId("kanban-card-overdue-t-1")).not.toBeInTheDocument();
  });

  it("uses formatUser for assignee display when provided", () => {
    const formatUser = (id: string): string => `User: ${id}`;
    render(<KanbanCard card={makeCard()} formatUser={formatUser} />);
    expect(screen.getByTestId("kanban-card-assignee-t-1")).toHaveTextContent("User: alice");
  });
});

describe("StageBadge", () => {
  it("renders badge with count", () => {
    render(<StageBadge stage="new_lead" count={5} />);
    expect(screen.getByTestId("stage-badge-new_lead")).toHaveTextContent("5");
  });

  it("renders badge for unknown stage with fallback color", () => {
    render(<StageBadge stage="unknown" count={0} />);
    expect(screen.getByTestId("stage-badge-unknown")).toBeInTheDocument();
  });
});
