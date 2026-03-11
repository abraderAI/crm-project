import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { Thread, Message } from "@/lib/api-types";
import type { ScoreBreakdown } from "@/lib/crm-types";
import { LeadDetail } from "./lead-detail";

function makeThread(overrides: Partial<Thread> = {}): Thread {
  return {
    id: "t-1",
    board_id: "b-1",
    title: "Lead: Acme Corp",
    body: "Enterprise lead from website form",
    slug: "lead-acme-corp",
    metadata: JSON.stringify({
      company: "Acme Corp",
      value: 75000,
      assigned_to: "alice",
      score: 82,
      contact_name: "Bob Smith",
      contact_email: "bob@acme.com",
      source: "website",
    }),
    author_id: "u-1",
    is_pinned: false,
    is_locked: false,
    is_hidden: false,
    vote_score: 5,
    status: "open",
    priority: "high",
    stage: "qualified",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
    ...overrides,
  };
}

const sampleMessage: Message = {
  id: "m-1",
  thread_id: "t-1",
  body: "Initial contact made.",
  author_id: "u-1",
  metadata: "{}",
  type: "note",
  created_at: "2025-01-02T00:00:00Z",
  updated_at: "2025-01-02T00:00:00Z",
};

const sampleBreakdown: ScoreBreakdown = {
  total: 82,
  rules: [
    { name: "has-email", description: "Has email", points: 20, matched: true },
    { name: "high-value", description: "Value > $10k", points: 30, matched: true },
  ],
};

describe("LeadDetail", () => {
  it("renders container", () => {
    render(<LeadDetail thread={makeThread()} messages={[]} />);
    expect(screen.getByTestId("lead-detail")).toBeInTheDocument();
  });

  it("renders lead title", () => {
    render(<LeadDetail thread={makeThread()} messages={[]} />);
    expect(screen.getByTestId("lead-detail-title")).toHaveTextContent("Lead: Acme Corp");
  });

  it("renders lead body", () => {
    render(<LeadDetail thread={makeThread()} messages={[]} />);
    expect(screen.getByTestId("lead-detail-body")).toHaveTextContent(
      "Enterprise lead from website form",
    );
  });

  it("hides body when absent", () => {
    render(<LeadDetail thread={makeThread({ body: undefined })} messages={[]} />);
    expect(screen.queryByTestId("lead-detail-body")).not.toBeInTheDocument();
  });

  it("renders stage pill", () => {
    render(<LeadDetail thread={makeThread()} messages={[]} />);
    expect(screen.getByTestId("lead-detail-stage")).toHaveTextContent("Qualified");
  });

  it("renders sidebar with lead info", () => {
    render(<LeadDetail thread={makeThread()} messages={[]} />);
    expect(screen.getByTestId("lead-sidebar")).toBeInTheDocument();
    expect(screen.getByTestId("lead-sidebar-company")).toHaveTextContent("Acme Corp");
    expect(screen.getByTestId("lead-sidebar-contact")).toHaveTextContent("Bob Smith");
    expect(screen.getByTestId("lead-sidebar-email")).toHaveTextContent("bob@acme.com");
    expect(screen.getByTestId("lead-sidebar-source")).toHaveTextContent("website");
    expect(screen.getByTestId("lead-sidebar-assignee")).toHaveTextContent("alice");
    expect(screen.getByTestId("lead-sidebar-value")).toHaveTextContent("$75,000");
    expect(screen.getByTestId("lead-sidebar-score")).toHaveTextContent("82");
    expect(screen.getByTestId("lead-sidebar-stage")).toHaveTextContent("Qualified");
  });

  it("renders thread details in sidebar", () => {
    render(<LeadDetail thread={makeThread()} messages={[]} />);
    expect(screen.getByTestId("lead-sidebar-status")).toHaveTextContent("open");
    expect(screen.getByTestId("lead-sidebar-priority")).toHaveTextContent("high");
    expect(screen.getByTestId("lead-sidebar-votes")).toHaveTextContent("5");
  });

  it("shows pinned/locked when applicable", () => {
    render(<LeadDetail thread={makeThread({ is_pinned: true, is_locked: true })} messages={[]} />);
    expect(screen.getByTestId("lead-sidebar-pinned")).toHaveTextContent("Yes");
    expect(screen.getByTestId("lead-sidebar-locked")).toHaveTextContent("Yes");
  });

  it("hides pinned/locked when not applicable", () => {
    render(<LeadDetail thread={makeThread()} messages={[]} />);
    expect(screen.queryByTestId("lead-sidebar-pinned")).not.toBeInTheDocument();
    expect(screen.queryByTestId("lead-sidebar-locked")).not.toBeInTheDocument();
  });

  it("renders activity count", () => {
    render(<LeadDetail thread={makeThread()} messages={[sampleMessage]} />);
    expect(screen.getByText("Activity (1)")).toBeInTheDocument();
  });

  it("renders message timeline", () => {
    render(<LeadDetail thread={makeThread()} messages={[sampleMessage]} />);
    expect(screen.getByTestId("message-item-m-1")).toBeInTheDocument();
  });

  it("renders enrichment section", () => {
    render(<LeadDetail thread={makeThread()} messages={[]} />);
    expect(screen.getByTestId("enrichment-section")).toBeInTheDocument();
  });

  it("shows enrichment empty state when no enrichment data", () => {
    render(<LeadDetail thread={makeThread()} messages={[]} />);
    expect(screen.getByTestId("enrichment-empty")).toBeInTheDocument();
  });

  it("shows enrichment content when metadata has enrichment", () => {
    const meta = JSON.stringify({
      company: "Acme",
      enrichment: { summary: "Great lead", next_action: "Call them" },
    });
    render(<LeadDetail thread={makeThread({ metadata: meta })} messages={[]} />);
    expect(screen.getByTestId("enrichment-content")).toBeInTheDocument();
    expect(screen.getByText("Great lead")).toBeInTheDocument();
  });

  it("renders score breakdown when provided", () => {
    render(<LeadDetail thread={makeThread()} messages={[]} scoreBreakdown={sampleBreakdown} />);
    expect(screen.getByTestId("score-breakdown")).toBeInTheDocument();
    expect(screen.getByTestId("score-total")).toHaveTextContent("82");
  });

  it("hides score breakdown when not provided", () => {
    render(<LeadDetail thread={makeThread()} messages={[]} />);
    expect(screen.queryByTestId("score-breakdown")).not.toBeInTheDocument();
  });

  it("renders enrich button when onEnrich provided", () => {
    render(<LeadDetail thread={makeThread()} messages={[]} onEnrich={vi.fn()} />);
    expect(screen.getByTestId("enrich-button")).toBeInTheDocument();
  });

  it("calls onEnrich when clicked", async () => {
    const user = userEvent.setup();
    const onEnrich = vi.fn();
    render(<LeadDetail thread={makeThread()} messages={[]} onEnrich={onEnrich} />);
    await user.click(screen.getByTestId("enrich-button"));
    expect(onEnrich).toHaveBeenCalledOnce();
  });

  it("shows customer org link for closed_won leads", () => {
    render(
      <LeadDetail
        thread={makeThread({ stage: "closed_won" })}
        messages={[]}
        customerOrgHref="/orgs/acme-customer"
      />,
    );
    const link = screen.getByTestId("lead-detail-customer-link");
    expect(link).toHaveAttribute("href", "/orgs/acme-customer");
    expect(link).toHaveTextContent("View customer organization");
  });

  it("hides customer org link for non-closed_won stages", () => {
    render(
      <LeadDetail
        thread={makeThread({ stage: "qualified" })}
        messages={[]}
        customerOrgHref="/orgs/acme-customer"
      />,
    );
    expect(screen.queryByTestId("lead-detail-customer-link")).not.toBeInTheDocument();
  });

  it("hides customer org link when no href", () => {
    render(<LeadDetail thread={makeThread({ stage: "closed_won" })} messages={[]} />);
    expect(screen.queryByTestId("lead-detail-customer-link")).not.toBeInTheDocument();
  });

  it("shows dash for missing status/priority", () => {
    render(
      <LeadDetail thread={makeThread({ status: undefined, priority: undefined })} messages={[]} />,
    );
    expect(screen.getByTestId("lead-sidebar-status")).toHaveTextContent("—");
    expect(screen.getByTestId("lead-sidebar-priority")).toHaveTextContent("—");
  });
});
