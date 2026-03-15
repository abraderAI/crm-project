import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ChannelHealthBadge } from "./channel-health-badge";

describe("ChannelHealthBadge", () => {
  it("renders the status text", () => {
    render(<ChannelHealthBadge status="healthy" />);
    expect(screen.getByTestId("channel-health-badge")).toHaveTextContent("healthy");
  });

  it("applies green colour for healthy status", () => {
    render(<ChannelHealthBadge status="healthy" />);
    expect(screen.getByTestId("channel-health-badge")).toHaveClass("bg-green-100");
  });

  it("applies yellow colour for degraded status", () => {
    render(<ChannelHealthBadge status="degraded" />);
    expect(screen.getByTestId("channel-health-badge")).toHaveClass("bg-yellow-100");
  });

  it("applies red colour for error status", () => {
    render(<ChannelHealthBadge status="error" />);
    expect(screen.getByTestId("channel-health-badge")).toHaveClass("bg-red-100");
  });

  it("applies grey colour for unconfigured status", () => {
    render(<ChannelHealthBadge status="unconfigured" />);
    expect(screen.getByTestId("channel-health-badge")).toHaveClass("bg-gray-100");
  });

  it("applies default colour for unknown status", () => {
    render(<ChannelHealthBadge status="unknown" />);
    expect(screen.getByTestId("channel-health-badge")).toHaveClass("bg-muted");
  });

  it("is case-insensitive for status matching", () => {
    render(<ChannelHealthBadge status="Healthy" />);
    expect(screen.getByTestId("channel-health-badge")).toHaveClass("bg-green-100");
  });

  it("renders as an inline element with badge styling", () => {
    render(<ChannelHealthBadge status="healthy" />);
    const badge = screen.getByTestId("channel-health-badge");
    expect(badge).toHaveClass("rounded-full");
    expect(badge).toHaveClass("text-xs");
    expect(badge).toHaveClass("font-medium");
  });
});
