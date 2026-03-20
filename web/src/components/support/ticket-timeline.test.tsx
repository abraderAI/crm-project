import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import type { SupportEntry } from "@/lib/api-types";
import { TicketTimeline } from "./ticket-timeline";

// Mock Clerk auth.
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: vi.fn().mockResolvedValue("tok") }),
}));

// Mock API calls.
vi.mock("@/lib/support-api", () => ({
  publishTicketEntry: vi.fn().mockResolvedValue({}),
  setEntryDeftVisibility: vi.fn().mockResolvedValue({}),
}));

const makeEntry = (overrides: Partial<SupportEntry> = {}): SupportEntry => ({
  id: "e1",
  thread_id: "t1",
  body: "<p>Hello</p>",
  author_id: "u1",
  metadata: "{}",
  type: "customer",
  is_deft_only: false,
  is_published: true,
  is_immutable: true,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
  ...overrides,
});

describe("TicketTimeline", () => {
  it("renders empty state when no entries", () => {
    render(<TicketTimeline entries={[]} ticketSlug="t1" isDeftMember={false} />);
    expect(screen.getByTestId("ticket-timeline-empty")).toBeInTheDocument();
  });

  it("renders entry with correct type badge", () => {
    const entry = makeEntry({ id: "e1", type: "agent_reply" });
    render(<TicketTimeline entries={[entry]} ticketSlug="t1" isDeftMember={false} />);
    expect(screen.getByTestId("entry-type-badge-e1")).toHaveTextContent("Agent Reply");
  });

  it("renders DEFT Only badge when is_deft_only is true", () => {
    const entry = makeEntry({ id: "e2", is_deft_only: true });
    render(<TicketTimeline entries={[entry]} ticketSlug="t1" isDeftMember={true} />);
    expect(screen.getByTestId("entry-deft-only-badge-e2")).toBeInTheDocument();
  });

  it("renders immutable lock icon when entry is immutable", () => {
    const entry = makeEntry({ id: "e3", is_immutable: true });
    render(<TicketTimeline entries={[entry]} ticketSlug="t1" isDeftMember={false} />);
    expect(screen.getByTestId("entry-immutable-icon-e3")).toBeInTheDocument();
  });

  it("shows publish button on draft entries for DEFT members", () => {
    const entry = makeEntry({ id: "e4", type: "draft", is_published: false, is_immutable: false });
    render(<TicketTimeline entries={[entry]} ticketSlug="t1" isDeftMember={true} />);
    expect(screen.getByTestId("publish-btn-e4")).toBeInTheDocument();
  });

  it("hides publish button for non-DEFT members", () => {
    const entry = makeEntry({ id: "e5", type: "draft", is_published: false, is_immutable: false });
    render(<TicketTimeline entries={[entry]} ticketSlug="t1" isDeftMember={false} />);
    expect(screen.queryByTestId("publish-btn-e5")).toBeNull();
  });

  it("shows deft-only toggle for DEFT members on any entry", () => {
    const entry = makeEntry({ id: "e6" });
    render(<TicketTimeline entries={[entry]} ticketSlug="t1" isDeftMember={true} />);
    expect(screen.getByTestId("deft-only-btn-e6")).toBeInTheDocument();
  });

  it("renders entry body HTML", () => {
    const entry = makeEntry({ id: "e7", body: "<p>Test body</p>" });
    render(<TicketTimeline entries={[entry]} ticketSlug="t1" isDeftMember={false} />);
    expect(screen.getByTestId("entry-body-e7")).toBeInTheDocument();
  });

  it("renders system_event badge correctly", () => {
    const entry = makeEntry({ id: "se1", type: "system_event" });
    render(<TicketTimeline entries={[entry]} ticketSlug="t1" isDeftMember={false} />);
    expect(screen.getByTestId("entry-type-badge-se1")).toHaveTextContent("System");
  });

  it("renders context badge correctly for DEFT members", () => {
    const entry = makeEntry({ id: "cx1", type: "context", is_deft_only: true });
    render(<TicketTimeline entries={[entry]} ticketSlug="t1" isDeftMember={true} />);
    expect(screen.getByTestId("entry-type-badge-cx1")).toHaveTextContent("Internal");
  });
});
