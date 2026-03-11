"use client";

import type { TypingUser } from "@/hooks/use-typing";

export interface TypingIndicatorProps {
  /** List of users currently typing. */
  typingUsers: TypingUser[];
}

/** Format a user-friendly typing message. */
export function formatTypingMessage(users: TypingUser[]): string {
  if (users.length === 0) return "";
  if (users.length === 1) return `${users[0]?.userName} is typing...`;
  if (users.length === 2) {
    return `${users[0]?.userName} and ${users[1]?.userName} are typing...`;
  }
  return `${users[0]?.userName} and ${users.length - 1} others are typing...`;
}

/** Displays a typing indicator when other users are typing. */
export function TypingIndicator({ typingUsers }: TypingIndicatorProps): React.ReactNode {
  if (typingUsers.length === 0) return null;

  const message = formatTypingMessage(typingUsers);

  return (
    <div
      className="flex items-center gap-2 py-2 text-xs text-muted-foreground"
      data-testid="typing-indicator"
      aria-live="polite"
    >
      <span className="flex gap-0.5" data-testid="typing-dots">
        <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-muted-foreground [animation-delay:-0.3s]" />
        <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-muted-foreground [animation-delay:-0.15s]" />
        <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-muted-foreground" />
      </span>
      <span data-testid="typing-message">{message}</span>
    </div>
  );
}
