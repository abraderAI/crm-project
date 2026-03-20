import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { EmailInboxList } from "./email-inbox-list";
import type { EmailInbox } from "@/lib/api-types";

const inbox1: EmailInbox = {
  id: "i1",
  org_id: "org1",
  name: "Support",
  email_address: "support@acme.com",
  imap_host: "imap.gmail.com",
  imap_port: 993,
  username: "support@acme.com",
  mailbox: "INBOX",
  routing_action: "support_ticket",
  enabled: true,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const inbox2: EmailInbox = {
  id: "i2",
  org_id: "org1",
  name: "Sales",
  email_address: "sales@acme.com",
  imap_host: "imap.gmail.com",
  imap_port: 993,
  username: "sales@acme.com",
  mailbox: "INBOX",
  routing_action: "sales_lead",
  enabled: false,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const noop = {
  onCreate: vi.fn().mockResolvedValue(inbox1),
  onUpdate: vi.fn().mockResolvedValue(inbox1),
  onDelete: vi.fn().mockResolvedValue(undefined),
};

describe("EmailInboxList", () => {
  it("renders the list container", () => {
    render(<EmailInboxList initialInboxes={[]} {...noop} />);
    expect(screen.getByTestId("email-inbox-list")).toBeInTheDocument();
    expect(screen.getByText("Email Inboxes")).toBeInTheDocument();
  });

  it("shows empty state when no inboxes", () => {
    render(<EmailInboxList initialInboxes={[]} {...noop} />);
    expect(screen.getByTestId("inbox-empty-state")).toBeInTheDocument();
    expect(screen.getByText("No email inboxes configured.")).toBeInTheDocument();
  });

  it("renders inbox rows when inboxes exist", () => {
    render(<EmailInboxList initialInboxes={[inbox1, inbox2]} {...noop} />);
    expect(screen.getByTestId(`inbox-row-${inbox1.id}`)).toBeInTheDocument();
    expect(screen.getByTestId(`inbox-row-${inbox2.id}`)).toBeInTheDocument();
    expect(screen.getByText("Support")).toBeInTheDocument();
    expect(screen.getByText("Sales")).toBeInTheDocument();
  });

  it("shows routing label in inbox row", () => {
    render(<EmailInboxList initialInboxes={[inbox1]} {...noop} />);
    expect(screen.getByText(/Support Ticket/)).toBeInTheDocument();
  });

  it("shows Add Inbox button", () => {
    render(<EmailInboxList initialInboxes={[]} {...noop} />);
    expect(screen.getByTestId("add-inbox-btn")).toBeInTheDocument();
  });

  it("opens form when Add Inbox is clicked", async () => {
    const user = userEvent.setup();
    render(<EmailInboxList initialInboxes={[]} {...noop} />);
    await user.click(screen.getByTestId("add-inbox-btn"));
    expect(screen.getByTestId("email-inbox-form")).toBeInTheDocument();
    // Empty state no longer shown while form is open
    expect(screen.queryByTestId("inbox-empty-state")).not.toBeInTheDocument();
  });

  it("opens form from empty state link", async () => {
    const user = userEvent.setup();
    render(<EmailInboxList initialInboxes={[]} {...noop} />);
    await user.click(screen.getByText("Add your first inbox"));
    expect(screen.getByTestId("email-inbox-form")).toBeInTheDocument();
  });

  it("closes form on cancel", async () => {
    const user = userEvent.setup();
    render(<EmailInboxList initialInboxes={[]} {...noop} />);
    await user.click(screen.getByTestId("add-inbox-btn"));
    await user.click(screen.getByTestId("inbox-cancel-btn"));
    expect(screen.queryByTestId("email-inbox-form")).not.toBeInTheDocument();
  });

  it("adds inbox to list on successful create", async () => {
    const user = userEvent.setup();
    const onCreate = vi.fn().mockResolvedValue(inbox1);
    render(
      <EmailInboxList
        initialInboxes={[]}
        onCreate={onCreate}
        onUpdate={noop.onUpdate}
        onDelete={noop.onDelete}
      />,
    );

    await user.click(screen.getByTestId("add-inbox-btn"));

    // Fill required fields then save.
    await user.type(screen.getByTestId("inbox-name"), "Support");
    await user.type(screen.getByTestId("inbox-imap-host"), "imap.gmail.com");
    await user.clear(screen.getByTestId("inbox-imap-port"));
    await user.type(screen.getByTestId("inbox-imap-port"), "993");
    await user.type(screen.getByTestId("inbox-username"), "support@acme.com");
    await user.type(screen.getByTestId("inbox-password"), "pass");
    await user.click(screen.getByTestId("inbox-save-btn"));

    expect(await screen.findByTestId(`inbox-row-${inbox1.id}`)).toBeInTheDocument();
    expect(screen.queryByTestId("email-inbox-form")).not.toBeInTheDocument();
  });

  it("opens edit form pre-filled when edit button clicked", async () => {
    const user = userEvent.setup();
    render(<EmailInboxList initialInboxes={[inbox1]} {...noop} />);
    await user.click(screen.getByTestId(`edit-inbox-${inbox1.id}`));
    expect(screen.getByTestId("email-inbox-form")).toBeInTheDocument();
    expect(screen.getByText("Edit Email Inbox")).toBeInTheDocument();
  });

  it("updates inbox row after successful edit", async () => {
    const user = userEvent.setup();
    const updatedInbox = { ...inbox1, name: "Renamed" };
    const onUpdate = vi.fn().mockResolvedValue(updatedInbox);
    render(
      <EmailInboxList
        initialInboxes={[inbox1]}
        onCreate={noop.onCreate}
        onUpdate={onUpdate}
        onDelete={noop.onDelete}
      />,
    );

    await user.click(screen.getByTestId(`edit-inbox-${inbox1.id}`));
    await user.click(screen.getByTestId("inbox-save-btn"));

    expect(await screen.findByText("Renamed")).toBeInTheDocument();
  });

  it("removes inbox from list after delete", async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn().mockResolvedValue(undefined);
    // Mock window.confirm to return true
    vi.stubGlobal("confirm", () => true);

    render(
      <EmailInboxList
        initialInboxes={[inbox1, inbox2]}
        onCreate={noop.onCreate}
        onUpdate={noop.onUpdate}
        onDelete={onDelete}
      />,
    );

    await user.click(screen.getByTestId(`delete-inbox-${inbox1.id}`));

    expect(await screen.findByTestId(`inbox-row-${inbox2.id}`)).toBeInTheDocument();
    expect(screen.queryByTestId(`inbox-row-${inbox1.id}`)).not.toBeInTheDocument();

    vi.unstubAllGlobals();
  });

  it("does not delete when confirm is cancelled", async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn();
    vi.stubGlobal("confirm", () => false);

    render(
      <EmailInboxList
        initialInboxes={[inbox1]}
        onCreate={noop.onCreate}
        onUpdate={noop.onUpdate}
        onDelete={onDelete}
      />,
    );
    await user.click(screen.getByTestId(`delete-inbox-${inbox1.id}`));

    expect(onDelete).not.toHaveBeenCalled();
    expect(screen.getByTestId(`inbox-row-${inbox1.id}`)).toBeInTheDocument();

    vi.unstubAllGlobals();
  });

  it("shows error message on delete failure", async () => {
    const user = userEvent.setup();
    vi.stubGlobal("confirm", () => true);
    const onDelete = vi.fn().mockRejectedValue(new Error("Delete failed"));

    render(
      <EmailInboxList
        initialInboxes={[inbox1]}
        onCreate={noop.onCreate}
        onUpdate={noop.onUpdate}
        onDelete={onDelete}
      />,
    );
    await user.click(screen.getByTestId(`delete-inbox-${inbox1.id}`));

    expect(await screen.findByTestId("inbox-list-error")).toHaveTextContent("Delete failed");

    vi.unstubAllGlobals();
  });
});
