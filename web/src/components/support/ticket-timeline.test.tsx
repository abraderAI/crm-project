import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, it, expect, vi } from "vitest";

import type { SupportEntry } from "@/lib/api-types";
import { TicketTimeline } from "./ticket-timeline";

vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: vi.fn().mockResolvedValue("tok") }),
}));

const mockPublishTicketEntry = vi.fn().mockResolvedValue({});
const mockSetEntryDeftVisibility = vi.fn().mockResolvedValue({});
const mockUpdateTicketEntry = vi.fn().mockResolvedValue({});
vi.mock("@/lib/support-api", () => ({
  publishTicketEntry: (...args: unknown[]) => mockPublishTicketEntry(...args),
  setEntryDeftVisibility: (...args: unknown[]) => mockSetEntryDeftVisibility(...args),
  updateTicketEntry: (...args: unknown[]) => mockUpdateTicketEntry(...args),
}));

function makeEntry(overrides: Partial<SupportEntry> = {}): SupportEntry {
  return {
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
  };
}

describe("TicketTimeline", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });
  it("renders empty state when there are no entries", () => {
    render(<TicketTimeline entries={[]} ticketSlug="t1" isDeftMember={false} />);
    expect(screen.getByTestId("ticket-timeline-empty")).toBeInTheDocument();
  });

  it("shows unhide toggle for DEFT members on non-internal DEFT-only entry", () => {
    const entry = makeEntry({ id: "e-visible", type: "agent_reply", is_deft_only: true });
    render(<TicketTimeline entries={[entry]} ticketSlug="t1" isDeftMember={true} />);
    expect(screen.getByTestId("deft-only-btn-e-visible")).toHaveTextContent("Unhide");
  });

  it("locks internal DEFT-only entries from unhide", () => {
    const entry = makeEntry({ id: "e-ctx", type: "context", is_deft_only: true });
    render(<TicketTimeline entries={[entry]} ticketSlug="t1" isDeftMember={true} />);
    expect(screen.queryByTestId("deft-only-btn-e-ctx")).not.toBeInTheDocument();
    expect(screen.getByTestId("deft-only-locked-e-ctx")).toHaveTextContent("Internal only");
  });

  it("shows publish button for own mutable draft for non-DEFT author", () => {
    const entry = makeEntry({
      id: "e-draft",
      type: "draft",
      author_id: "u1",
      is_immutable: false,
      is_published: false,
    });
    render(
      <TicketTimeline entries={[entry]} ticketSlug="t1" isDeftMember={false} currentUserId="u1" />,
    );
    expect(screen.getByTestId("publish-btn-e-draft")).toBeInTheDocument();
  });

  it("shows hide toggle for DEFT members on requestor-visible entry", () => {
    const entry = makeEntry({ id: "e-hide", type: "customer", is_deft_only: false });
    render(<TicketTimeline entries={[entry]} ticketSlug="t1" isDeftMember={true} />);
    expect(screen.getByTestId("deft-only-btn-e-hide")).toHaveTextContent("Hide");
  });

  it("does not render DEFT visibility controls for non-DEFT viewers", () => {
    const entry = makeEntry({ id: "e-public", type: "customer", is_deft_only: false });
    render(<TicketTimeline entries={[entry]} ticketSlug="t1" isDeftMember={false} />);
    expect(screen.queryByTestId("deft-only-btn-e-public")).not.toBeInTheDocument();
    expect(screen.queryByTestId("deft-only-locked-e-public")).not.toBeInTheDocument();
  });

  it("renders structured author name and org badge when resolveUser is provided", () => {
    const entry = makeEntry({ id: "e-resolved", author_id: "u1" });
    const resolveUser = (userId: string) =>
      userId === "u1" ? { display_name: "Alice", org_name: "Acme Corp" } : undefined;
    render(
      <TicketTimeline
        entries={[entry]}
        ticketSlug="t1"
        isDeftMember={false}
        resolveUser={resolveUser}
      />,
    );
    const authorEl = screen.getByTestId("entry-author-e-resolved");
    expect(authorEl).toHaveTextContent("Alice");
    expect(screen.getByTestId("entry-org-badge-e-resolved")).toHaveTextContent("Acme Corp");
  });

  it("falls back to formatUser when resolveUser returns undefined", () => {
    const entry = makeEntry({ id: "e-fallback", author_id: "u-unknown" });
    const resolveUser = () => undefined;
    const formatUser = (userId: string) => `User:${userId}`;
    render(
      <TicketTimeline
        entries={[entry]}
        ticketSlug="t1"
        isDeftMember={false}
        resolveUser={resolveUser}
        formatUser={formatUser}
      />,
    );
    expect(screen.getByTestId("entry-author-e-fallback")).toHaveTextContent("User:u-unknown");
  });

  it("publishes a draft when send is clicked", async () => {
    const user = userEvent.setup();
    const entry = makeEntry({
      id: "e-publish",
      type: "draft",
      author_id: "u1",
      is_immutable: false,
      is_published: false,
    });
    render(
      <TicketTimeline
        entries={[entry]}
        ticketSlug="ticket-1"
        isDeftMember={false}
        currentUserId="u1"
      />,
    );
    await user.click(screen.getByTestId("publish-btn-e-publish"));
    await waitFor(() => {
      expect(mockPublishTicketEntry).toHaveBeenCalledWith("tok", "ticket-1", "e-publish");
    });
  });

  it("toggles DEFT visibility on hide click", async () => {
    const user = userEvent.setup();
    const entry = makeEntry({ id: "e-toggle", type: "customer", is_deft_only: false });
    render(<TicketTimeline entries={[entry]} ticketSlug="ticket-1" isDeftMember={true} />);
    await user.click(screen.getByTestId("deft-only-btn-e-toggle"));
    await waitFor(() => {
      expect(mockSetEntryDeftVisibility).toHaveBeenCalledWith("tok", "ticket-1", "e-toggle", true);
    });
  });

  it("shows timeline error when publish fails", async () => {
    const user = userEvent.setup();
    mockPublishTicketEntry.mockRejectedValueOnce(new Error("publish failed"));
    const entry = makeEntry({
      id: "e-err",
      type: "draft",
      author_id: "u1",
      is_immutable: false,
      is_published: false,
    });
    render(
      <TicketTimeline
        entries={[entry]}
        ticketSlug="ticket-1"
        isDeftMember={false}
        currentUserId="u1"
      />,
    );
    await user.click(screen.getByTestId("publish-btn-e-err"));
    expect(await screen.findByTestId("timeline-error")).toHaveTextContent("publish failed");
  });
});
