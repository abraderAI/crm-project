import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { VoteButton } from "./vote-button";

describe("VoteButton", () => {
  it("renders the current vote score", () => {
    render(<VoteButton voteScore={42} hasVoted={false} onToggle={vi.fn()} />);
    expect(screen.getByTestId("vote-score")).toHaveTextContent("42");
  });

  it("renders not-voted state by default", () => {
    render(<VoteButton voteScore={5} hasVoted={false} onToggle={vi.fn()} />);
    const button = screen.getByTestId("vote-button");
    expect(button).not.toHaveClass("border-primary");
  });

  it("renders voted state when hasVoted is true", () => {
    render(<VoteButton voteScore={5} hasVoted={true} onToggle={vi.fn()} />);
    const button = screen.getByTestId("vote-button");
    expect(button).toHaveClass("border-primary");
  });

  it("calls onToggle when clicked", async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    render(<VoteButton voteScore={5} hasVoted={false} onToggle={onToggle} />);

    await user.click(screen.getByTestId("vote-button"));
    expect(onToggle).toHaveBeenCalledOnce();
  });

  it("optimistically increments score on vote", async () => {
    const user = userEvent.setup();
    render(<VoteButton voteScore={10} hasVoted={false} onToggle={vi.fn()} />);

    await user.click(screen.getByTestId("vote-button"));
    expect(screen.getByTestId("vote-score")).toHaveTextContent("11");
  });

  it("optimistically decrements score on unvote", async () => {
    const user = userEvent.setup();
    render(<VoteButton voteScore={10} hasVoted={true} onToggle={vi.fn()} />);

    await user.click(screen.getByTestId("vote-button"));
    expect(screen.getByTestId("vote-score")).toHaveTextContent("9");
  });

  it("applies custom weight on vote", async () => {
    const user = userEvent.setup();
    render(<VoteButton voteScore={10} hasVoted={false} userWeight={3} onToggle={vi.fn()} />);

    await user.click(screen.getByTestId("vote-button"));
    expect(screen.getByTestId("vote-score")).toHaveTextContent("13");
  });

  it("applies custom weight on unvote", async () => {
    const user = userEvent.setup();
    render(<VoteButton voteScore={10} hasVoted={true} userWeight={3} onToggle={vi.fn()} />);

    await user.click(screen.getByTestId("vote-button"));
    expect(screen.getByTestId("vote-score")).toHaveTextContent("7");
  });

  it("shows vote weight in title attribute", () => {
    render(<VoteButton voteScore={5} hasVoted={false} userWeight={2} onToggle={vi.fn()} />);
    expect(screen.getByTestId("vote-button")).toHaveAttribute("title", "Vote weight: 2");
  });

  it("does not call onToggle when disabled", async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    render(<VoteButton voteScore={5} hasVoted={false} onToggle={onToggle} disabled={true} />);

    await user.click(screen.getByTestId("vote-button"));
    expect(onToggle).not.toHaveBeenCalled();
  });

  it("applies disabled styling", () => {
    render(<VoteButton voteScore={5} hasVoted={false} onToggle={vi.fn()} disabled={true} />);
    expect(screen.getByTestId("vote-button")).toBeDisabled();
  });

  it("toggles visual state on double click", async () => {
    const user = userEvent.setup();
    render(<VoteButton voteScore={10} hasVoted={false} onToggle={vi.fn()} />);

    await user.click(screen.getByTestId("vote-button"));
    expect(screen.getByTestId("vote-score")).toHaveTextContent("11");

    await user.click(screen.getByTestId("vote-button"));
    expect(screen.getByTestId("vote-score")).toHaveTextContent("10");
  });

  it("renders the vote icon", () => {
    render(<VoteButton voteScore={0} hasVoted={false} onToggle={vi.fn()} />);
    expect(screen.getByTestId("vote-icon")).toBeInTheDocument();
  });
});
