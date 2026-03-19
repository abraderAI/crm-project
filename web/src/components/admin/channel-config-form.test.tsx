import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ChannelConfigForm } from "./channel-config-form";
import type { ChannelConfig } from "@/lib/api-types";

const emailConfig: ChannelConfig = {
  id: "cfg1",
  org_id: "org1",
  channel_type: "email",
  settings: JSON.stringify({
    imap_host: "mail.example.com",
    imap_port: 993,
    username: "user@example.com",
    password: "secret123",
    mailbox: "INBOX",
  }),
  enabled: true,
};

const voiceConfig: ChannelConfig = {
  id: "cfg2",
  org_id: "org1",
  channel_type: "voice",
  settings: JSON.stringify({
    livekit_project_url: "wss://lk.example.com",
    livekit_api_key: "key123",
    livekit_api_secret: "secret456",
  }),
  enabled: false,
};

describe("ChannelConfigForm", () => {
  it("renders the form with channel type heading", () => {
    render(<ChannelConfigForm channelType="email" initialConfig={null} onSave={vi.fn()} />);
    expect(screen.getByTestId("channel-config-form")).toBeInTheDocument();
    expect(screen.getByText("Email Configuration")).toBeInTheDocument();
  });

  it("renders voice configuration heading", () => {
    render(<ChannelConfigForm channelType="voice" initialConfig={null} onSave={vi.fn()} />);
    expect(screen.getByText("Voice Configuration")).toBeInTheDocument();
  });

  it("renders chat configuration heading", () => {
    render(<ChannelConfigForm channelType="chat" initialConfig={null} onSave={vi.fn()} />);
    expect(screen.getByText("Chat Configuration")).toBeInTheDocument();
  });

  it("renders email fields", () => {
    render(<ChannelConfigForm channelType="email" initialConfig={null} onSave={vi.fn()} />);
    expect(screen.getByTestId("field-input-imap_host")).toBeInTheDocument();
    expect(screen.getByTestId("field-input-imap_port")).toBeInTheDocument();
    expect(screen.getByTestId("field-input-username")).toBeInTheDocument();
    expect(screen.getByTestId("field-input-password")).toBeInTheDocument();
    expect(screen.getByTestId("field-input-mailbox")).toBeInTheDocument();
  });

  it("renders voice fields", () => {
    render(<ChannelConfigForm channelType="voice" initialConfig={null} onSave={vi.fn()} />);
    expect(screen.getByTestId("field-input-livekit_project_url")).toBeInTheDocument();
    expect(screen.getByTestId("field-input-livekit_api_key")).toBeInTheDocument();
    expect(screen.getByTestId("field-input-livekit_api_secret")).toBeInTheDocument();
  });

  it("renders chat fields", () => {
    render(<ChannelConfigForm channelType="chat" initialConfig={null} onSave={vi.fn()} />);
    expect(screen.getByTestId("field-input-jwt_secret")).toBeInTheDocument();
    expect(screen.getByTestId("field-input-allowed_origins")).toBeInTheDocument();
    expect(screen.getByTestId("field-input-max_session_minutes")).toBeInTheDocument();
  });

  it("populates non-masked fields from initial config", () => {
    render(<ChannelConfigForm channelType="email" initialConfig={emailConfig} onSave={vi.fn()} />);
    expect(screen.getByTestId("field-input-imap_host")).toHaveValue("mail.example.com");
    expect(screen.getByTestId("field-input-mailbox")).toHaveValue("INBOX");
  });

  it("does not populate masked fields from initial config", () => {
    render(<ChannelConfigForm channelType="email" initialConfig={emailConfig} onSave={vi.fn()} />);
    expect(screen.getByTestId("field-input-password")).toHaveValue("");
  });

  it("shows masked indicator for existing secrets", () => {
    render(<ChannelConfigForm channelType="email" initialConfig={emailConfig} onSave={vi.fn()} />);
    expect(screen.getByTestId("field-masked-password")).toHaveTextContent("••••••••");
  });

  it("does not show masked indicator when no existing secret", () => {
    render(<ChannelConfigForm channelType="email" initialConfig={null} onSave={vi.fn()} />);
    expect(screen.queryByTestId("field-masked-password")).not.toBeInTheDocument();
  });

  it("renders enabled toggle", () => {
    render(<ChannelConfigForm channelType="email" initialConfig={emailConfig} onSave={vi.fn()} />);
    const toggle = screen.getByTestId("channel-enabled-toggle");
    expect(toggle).toHaveAttribute("aria-checked", "true");
  });

  it("toggles enabled state", async () => {
    const user = userEvent.setup();
    render(<ChannelConfigForm channelType="email" initialConfig={emailConfig} onSave={vi.fn()} />);
    const toggle = screen.getByTestId("channel-enabled-toggle");
    expect(toggle).toHaveAttribute("aria-checked", "true");
    await user.click(toggle);
    expect(toggle).toHaveAttribute("aria-checked", "false");
  });

  it("calls onSave with settings and enabled state", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    render(<ChannelConfigForm channelType="email" initialConfig={null} onSave={onSave} />);
    await user.type(screen.getByTestId("field-input-imap_host"), "mail.test.com");
    await user.click(screen.getByTestId("config-save-btn"));
    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({ imap_host: "mail.test.com" }),
      false,
    );
  });

  it("shows saving state on save button", async () => {
    const user = userEvent.setup();
    let resolvePromise: (() => void) | undefined;
    const onSave = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolvePromise = resolve;
        }),
    );
    render(<ChannelConfigForm channelType="email" initialConfig={null} onSave={onSave} />);
    await user.click(screen.getByTestId("config-save-btn"));
    expect(screen.getByTestId("config-save-btn")).toHaveTextContent("Saving...");
    resolvePromise?.();
  });

  it("shows error on save failure", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockRejectedValue(new Error("Network error"));
    render(<ChannelConfigForm channelType="email" initialConfig={null} onSave={onSave} />);
    await user.click(screen.getByTestId("config-save-btn"));
    expect(await screen.findByTestId("config-error")).toHaveTextContent("Network error");
  });

  it("resets form to initial values", async () => {
    const user = userEvent.setup();
    render(<ChannelConfigForm channelType="email" initialConfig={emailConfig} onSave={vi.fn()} />);
    const input = screen.getByTestId("field-input-imap_host");
    await user.clear(input);
    await user.type(input, "changed.com");
    expect(input).toHaveValue("changed.com");
    await user.click(screen.getByTestId("config-reset-btn"));
    expect(input).toHaveValue("mail.example.com");
  });

  it("renders save and reset buttons", () => {
    render(<ChannelConfigForm channelType="voice" initialConfig={voiceConfig} onSave={vi.fn()} />);
    expect(screen.getByTestId("config-save-btn")).toBeInTheDocument();
    expect(screen.getByTestId("config-reset-btn")).toBeInTheDocument();
  });

  it("sets enabled toggle to false for disabled config", () => {
    render(<ChannelConfigForm channelType="voice" initialConfig={voiceConfig} onSave={vi.fn()} />);
    const toggle = screen.getByTestId("channel-enabled-toggle");
    expect(toggle).toHaveAttribute("aria-checked", "false");
  });
});
