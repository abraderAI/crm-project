import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ChannelOverview } from "./channel-overview";
import type { ChannelHealth, ChannelType } from "@/lib/api-types";

const healthyEmail: ChannelHealth = {
  channel_type: "email",
  enabled: true,
  last_event_at: "2026-03-14T12:00:00Z",
  error_rate: 0.02,
  status: "healthy",
};

const degradedVoice: ChannelHealth = {
  channel_type: "voice",
  enabled: true,
  last_event_at: "2026-03-14T11:00:00Z",
  error_rate: 0.15,
  status: "degraded",
};

const errorChat: ChannelHealth = {
  channel_type: "chat",
  enabled: false,
  last_event_at: "",
  error_rate: 0.5,
  status: "error",
};

const healthMap: Record<ChannelType, ChannelHealth | null> = {
  email: healthyEmail,
  voice: degradedVoice,
  chat: errorChat,
};

const emptyHealthMap: Record<ChannelType, ChannelHealth | null> = {
  email: null,
  voice: null,
  chat: null,
};

describe("ChannelOverview", () => {
  it("renders the heading", () => {
    render(<ChannelOverview healthMap={healthMap} />);
    expect(screen.getByText("IO Channels")).toBeInTheDocument();
  });

  it("shows loading state", () => {
    render(<ChannelOverview healthMap={emptyHealthMap} loading={true} />);
    expect(screen.getByTestId("channel-overview-loading")).toBeInTheDocument();
  });

  it("renders 3 channel cards", () => {
    render(<ChannelOverview healthMap={healthMap} />);
    expect(screen.getByTestId("channel-card-email")).toBeInTheDocument();
    expect(screen.getByTestId("channel-card-voice")).toBeInTheDocument();
    expect(screen.getByTestId("channel-card-chat")).toBeInTheDocument();
  });

  it("shows channel names", () => {
    render(<ChannelOverview healthMap={healthMap} />);
    expect(screen.getByText("Email")).toBeInTheDocument();
    expect(screen.getByText("Voice")).toBeInTheDocument();
    expect(screen.getByText("Chat")).toBeInTheDocument();
  });

  it("shows enabled badge for enabled channels", () => {
    render(<ChannelOverview healthMap={healthMap} />);
    expect(screen.getByTestId("channel-enabled-email")).toHaveTextContent("Enabled");
  });

  it("shows disabled badge for disabled channels", () => {
    render(<ChannelOverview healthMap={healthMap} />);
    expect(screen.getByTestId("channel-enabled-chat")).toHaveTextContent("Disabled");
  });

  it("shows error rate", () => {
    render(<ChannelOverview healthMap={healthMap} />);
    expect(screen.getByTestId("channel-error-rate-email")).toHaveTextContent("2.0%");
  });

  it("shows N/A for error rate when no health data", () => {
    render(<ChannelOverview healthMap={emptyHealthMap} />);
    expect(screen.getByTestId("channel-error-rate-email")).toHaveTextContent("N/A");
  });

  it("shows Never for last event when no health data", () => {
    render(<ChannelOverview healthMap={emptyHealthMap} />);
    expect(screen.getByTestId("channel-last-event-email")).toHaveTextContent("Never");
  });

  it("renders Configure buttons with correct hrefs", () => {
    render(<ChannelOverview healthMap={healthMap} />);
    const configLink = screen.getByTestId("channel-configure-email");
    expect(configLink).toHaveAttribute("href", "/admin/channels/email");
  });

  it("renders DLQ buttons", () => {
    render(<ChannelOverview healthMap={healthMap} />);
    expect(screen.getByTestId("channel-dlq-email")).toBeInTheDocument();
  });

  it("shows DLQ badge count when > 0", () => {
    render(<ChannelOverview healthMap={healthMap} dlqCounts={{ email: 5, voice: 0, chat: 0 }} />);
    expect(screen.getByTestId("channel-dlq-count-email")).toHaveTextContent("5");
  });

  it("does not show DLQ badge count when 0", () => {
    render(<ChannelOverview healthMap={healthMap} dlqCounts={{ email: 0, voice: 0, chat: 0 }} />);
    expect(screen.queryByTestId("channel-dlq-count-email")).not.toBeInTheDocument();
  });

  it("renders channel grid container", () => {
    render(<ChannelOverview healthMap={healthMap} />);
    expect(screen.getByTestId("channel-grid")).toBeInTheDocument();
  });

  it("renders unconfigured status for null health", () => {
    render(<ChannelOverview healthMap={emptyHealthMap} />);
    const badges = screen.getAllByTestId("channel-health-badge");
    expect(badges[0]).toHaveTextContent("unconfigured");
  });
});
