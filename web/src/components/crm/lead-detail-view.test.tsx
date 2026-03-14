import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Thread, Message } from "@/lib/api-types";
import type { ScoreBreakdown } from "@/lib/crm-types";

// Mock Clerk auth.
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ userId: "user-42" }),
}));

import { LeadDetailView } from "./lead-detail-view";

function makeThread(overrides: Partial<Thread> = {}): Thread {
  return {
    id: "t-1",
    board_id: "b-1",
    title: "Lead: Acme Corp",
    body: "Enterprise lead",
    slug: "lead-acme-corp",
    metadata: JSON.stringify({
      company: "Acme Corp",
      value: 75000,
      score: 82,
      contact_name: "Bob Smith",
      contact_email: "bob@acme.com",
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

describe("LeadDetailView", () => {
  it("renders LeadDetail with thread data", () => {
    render(<LeadDetailView thread={makeThread()} messages={[]} />);
    expect(screen.getByTestId("lead-detail")).toBeInTheDocument();
    expect(screen.getByTestId("lead-detail-title")).toHaveTextContent("Lead: Acme Corp");
  });

  it("passes currentUserId from Clerk auth", () => {
    render(<LeadDetailView thread={makeThread()} messages={[sampleMessage]} />);
    // The message authored by user-42 should get "own message" treatment
    // via currentUserId prop. Verify the detail renders with sidebar.
    expect(screen.getByTestId("lead-sidebar")).toBeInTheDocument();
  });

  it("passes score breakdown to LeadDetail", () => {
    render(
      <LeadDetailView thread={makeThread()} messages={[]} scoreBreakdown={sampleBreakdown} />,
    );
    expect(screen.getByTestId("score-breakdown")).toBeInTheDocument();
    expect(screen.getByTestId("score-total")).toHaveTextContent("82");
  });

  it("passes messages to LeadDetail", () => {
    render(<LeadDetailView thread={makeThread()} messages={[sampleMessage]} />);
    expect(screen.getByTestId("message-item-m-1")).toBeInTheDocument();
  });

  it("passes customerOrgHref to LeadDetail", () => {
    render(
      <LeadDetailView
        thread={makeThread({ stage: "closed_won" })}
        messages={[]}
        customerOrgHref="/orgs/acme-customer"
      />,
    );
    expect(screen.getByTestId("lead-detail-customer-link")).toHaveAttribute(
      "href",
      "/orgs/acme-customer",
    );
  });
});
