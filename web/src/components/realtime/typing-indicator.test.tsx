import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { TypingIndicator, formatTypingMessage } from "./typing-indicator";
import type { TypingUser } from "@/hooks/use-typing";

const makeUser = (id: string, name: string): TypingUser => ({
  userId: id,
  userName: name,
  expiresAt: Date.now() + 5000,
});

describe("formatTypingMessage", () => {
  it("returns empty string for no users", () => {
    expect(formatTypingMessage([])).toBe("");
  });

  it("formats single user", () => {
    expect(formatTypingMessage([makeUser("1", "Alice")])).toBe("Alice is typing...");
  });

  it("formats two users", () => {
    expect(formatTypingMessage([makeUser("1", "Alice"), makeUser("2", "Bob")])).toBe(
      "Alice and Bob are typing...",
    );
  });

  it("formats three+ users", () => {
    expect(
      formatTypingMessage([makeUser("1", "Alice"), makeUser("2", "Bob"), makeUser("3", "Eve")]),
    ).toBe("Alice and 2 others are typing...");
  });
});

describe("TypingIndicator", () => {
  it("renders nothing when no users are typing", () => {
    const { container } = render(<TypingIndicator typingUsers={[]} />);
    expect(container.innerHTML).toBe("");
  });

  it("renders indicator for one user", () => {
    render(<TypingIndicator typingUsers={[makeUser("1", "Alice")]} />);
    expect(screen.getByTestId("typing-indicator")).toBeInTheDocument();
    expect(screen.getByTestId("typing-message")).toHaveTextContent("Alice is typing...");
  });

  it("renders typing dots", () => {
    render(<TypingIndicator typingUsers={[makeUser("1", "Alice")]} />);
    expect(screen.getByTestId("typing-dots")).toBeInTheDocument();
    // Three animated dots.
    expect(screen.getByTestId("typing-dots").children).toHaveLength(3);
  });

  it("renders indicator for multiple users", () => {
    render(<TypingIndicator typingUsers={[makeUser("1", "Alice"), makeUser("2", "Bob")]} />);
    expect(screen.getByTestId("typing-message")).toHaveTextContent("Alice and Bob are typing...");
  });

  it("has aria-live for accessibility", () => {
    render(<TypingIndicator typingUsers={[makeUser("1", "Alice")]} />);
    expect(screen.getByTestId("typing-indicator")).toHaveAttribute("aria-live", "polite");
  });
});
