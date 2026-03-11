import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { ScoreBreakdown } from "@/lib/crm-types";
import { ScoreBreakdownView } from "./score-breakdown";

const fullBreakdown: ScoreBreakdown = {
  total: 85,
  rules: [
    { name: "has-email", description: "Contact has email", points: 20, matched: true },
    { name: "high-value", description: "Deal value > $10k", points: 30, matched: true },
    { name: "enterprise", description: "Enterprise company", points: 25, matched: true },
    { name: "recent-activity", description: "Activity in last 7 days", points: 10, matched: false },
  ],
};

describe("ScoreBreakdownView", () => {
  it("renders container", () => {
    render(<ScoreBreakdownView breakdown={fullBreakdown} />);
    expect(screen.getByTestId("score-breakdown")).toBeInTheDocument();
  });

  it("renders total score", () => {
    render(<ScoreBreakdownView breakdown={fullBreakdown} />);
    expect(screen.getByTestId("score-total")).toHaveTextContent("85");
  });

  it("renders matched rules section", () => {
    render(<ScoreBreakdownView breakdown={fullBreakdown} />);
    expect(screen.getByTestId("score-matched-rules")).toBeInTheDocument();
  });

  it("renders matched rules with names", () => {
    render(<ScoreBreakdownView breakdown={fullBreakdown} />);
    expect(screen.getByTestId("score-rule-has-email")).toBeInTheDocument();
    expect(screen.getByTestId("score-rule-high-value")).toBeInTheDocument();
    expect(screen.getByTestId("score-rule-enterprise")).toBeInTheDocument();
  });

  it("shows points for each rule", () => {
    render(<ScoreBreakdownView breakdown={fullBreakdown} />);
    expect(screen.getByTestId("score-rule-points-has-email")).toHaveTextContent("+20");
    expect(screen.getByTestId("score-rule-points-high-value")).toHaveTextContent("+30");
  });

  it("renders unmatched rules section", () => {
    render(<ScoreBreakdownView breakdown={fullBreakdown} />);
    expect(screen.getByTestId("score-unmatched-rules")).toBeInTheDocument();
    expect(screen.getByTestId("score-rule-recent-activity")).toBeInTheDocument();
  });

  it("hides matched section when no matched rules", () => {
    const breakdown: ScoreBreakdown = {
      total: 0,
      rules: [{ name: "r1", description: "d1", points: 10, matched: false }],
    };
    render(<ScoreBreakdownView breakdown={breakdown} />);
    expect(screen.queryByTestId("score-matched-rules")).not.toBeInTheDocument();
  });

  it("hides unmatched section when all matched", () => {
    const breakdown: ScoreBreakdown = {
      total: 50,
      rules: [{ name: "r1", description: "d1", points: 50, matched: true }],
    };
    render(<ScoreBreakdownView breakdown={breakdown} />);
    expect(screen.queryByTestId("score-unmatched-rules")).not.toBeInTheDocument();
  });

  it("shows empty state when no rules", () => {
    render(<ScoreBreakdownView breakdown={{ total: 0, rules: [] }} />);
    expect(screen.getByTestId("score-no-rules")).toHaveTextContent("No scoring rules configured.");
  });

  it("shows rule descriptions", () => {
    render(<ScoreBreakdownView breakdown={fullBreakdown} />);
    expect(screen.getByText("Contact has email")).toBeInTheDocument();
    expect(screen.getByText("Deal value > $10k")).toBeInTheDocument();
  });
});
