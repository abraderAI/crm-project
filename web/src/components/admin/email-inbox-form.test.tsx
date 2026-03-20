import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { EmailInboxForm } from "./email-inbox-form";
import type { EmailInbox } from "@/lib/api-types";

const existingInbox: EmailInbox = {
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

describe("EmailInboxForm", () => {
  it("renders Add Email Inbox heading for new inbox", () => {
    render(<EmailInboxForm onSave={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.getByTestId("email-inbox-form")).toBeInTheDocument();
    expect(screen.getByText("Add Email Inbox")).toBeInTheDocument();
  });

  it("renders Edit Email Inbox heading when editing", () => {
    render(<EmailInboxForm inbox={existingInbox} onSave={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.getByText("Edit Email Inbox")).toBeInTheDocument();
  });

  it("pre-fills fields from existing inbox", () => {
    render(<EmailInboxForm inbox={existingInbox} onSave={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.getByTestId("inbox-name")).toHaveValue("Support");
    expect(screen.getByTestId("inbox-email-address")).toHaveValue("support@acme.com");
    expect(screen.getByTestId("inbox-imap-host")).toHaveValue("imap.gmail.com");
    expect(screen.getByTestId("inbox-imap-port")).toHaveValue(993);
    expect(screen.getByTestId("inbox-username")).toHaveValue("support@acme.com");
    expect(screen.getByTestId("inbox-mailbox")).toHaveValue("INBOX");
  });

  it("password field is always empty (never pre-filled)", () => {
    render(<EmailInboxForm inbox={existingInbox} onSave={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.getByTestId("inbox-password")).toHaveValue("");
  });

  it("shows password required indicator for new inbox", () => {
    render(<EmailInboxForm onSave={vi.fn()} onCancel={vi.fn()} />);
    // New inbox: password field has required attribute
    expect(screen.getByTestId("inbox-password")).toBeRequired();
  });

  it("password is not required when editing", () => {
    render(<EmailInboxForm inbox={existingInbox} onSave={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.getByTestId("inbox-password")).not.toBeRequired();
  });

  it("renders routing action selector with all options", () => {
    render(<EmailInboxForm onSave={vi.fn()} onCancel={vi.fn()} />);
    const select = screen.getByTestId("inbox-routing-action");
    expect(select).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Support Ticket" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Sales Lead" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "General Message" })).toBeInTheDocument();
  });

  it("shows routing description below selector", () => {
    render(<EmailInboxForm onSave={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.getByText("Creates a ticket in the Support space")).toBeInTheDocument();
  });

  it("updates routing description when selection changes", async () => {
    const user = userEvent.setup();
    render(<EmailInboxForm onSave={vi.fn()} onCancel={vi.fn()} />);
    await user.selectOptions(screen.getByTestId("inbox-routing-action"), "sales_lead");
    expect(screen.getByText("Creates a lead in the CRM space")).toBeInTheDocument();
  });

  it("renders enabled toggle defaulting to true for new inbox", () => {
    render(<EmailInboxForm onSave={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.getByTestId("inbox-enabled-toggle")).toHaveAttribute("aria-checked", "true");
  });

  it("toggles enabled state", async () => {
    const user = userEvent.setup();
    render(<EmailInboxForm onSave={vi.fn()} onCancel={vi.fn()} />);
    const toggle = screen.getByTestId("inbox-enabled-toggle");
    await user.click(toggle);
    expect(toggle).toHaveAttribute("aria-checked", "false");
  });

  it("calls onSave with input data on submit", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockResolvedValue(undefined);
    render(<EmailInboxForm onSave={onSave} onCancel={vi.fn()} />);

    await user.type(screen.getByTestId("inbox-name"), "Support");
    await user.type(screen.getByTestId("inbox-imap-host"), "imap.gmail.com");
    await user.clear(screen.getByTestId("inbox-imap-port"));
    await user.type(screen.getByTestId("inbox-imap-port"), "993");
    await user.type(screen.getByTestId("inbox-username"), "support@acme.com");
    await user.type(screen.getByTestId("inbox-password"), "app-password");

    await user.click(screen.getByTestId("inbox-save-btn"));

    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        name: "Support",
        imap_host: "imap.gmail.com",
        username: "support@acme.com",
        password: "app-password",
        routing_action: "support_ticket",
        enabled: true,
      }),
    );
  });

  it("omits password from input when field is blank on edit", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockResolvedValue(undefined);
    render(<EmailInboxForm inbox={existingInbox} onSave={onSave} onCancel={vi.fn()} />);

    await user.click(screen.getByTestId("inbox-save-btn"));

    const call = onSave.mock.calls[0]?.[0] as Record<string, unknown>;
    expect(call.password).toBeUndefined();
  });

  it("shows saving state while submitting", async () => {
    const user = userEvent.setup();
    let resolve: (() => void) | undefined;
    const onSave = vi.fn(
      () =>
        new Promise<void>((r) => {
          resolve = r;
        }),
    );
    render(<EmailInboxForm inbox={existingInbox} onSave={onSave} onCancel={vi.fn()} />);
    await user.click(screen.getByTestId("inbox-save-btn"));
    expect(screen.getByTestId("inbox-save-btn")).toHaveTextContent("Saving…");
    resolve?.();
  });

  it("shows error message on save failure", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockRejectedValue(new Error("Server error"));
    render(<EmailInboxForm inbox={existingInbox} onSave={onSave} onCancel={vi.fn()} />);
    await user.click(screen.getByTestId("inbox-save-btn"));
    expect(await screen.findByTestId("inbox-form-error")).toHaveTextContent("Server error");
  });

  it("calls onCancel when Cancel button is clicked", async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    render(<EmailInboxForm onSave={vi.fn()} onCancel={onCancel} />);
    await user.click(screen.getByTestId("inbox-cancel-btn"));
    expect(onCancel).toHaveBeenCalled();
  });
});
