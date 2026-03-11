"use client";

import { useState } from "react";
import { ThumbsUp } from "lucide-react";
import { cn } from "@/lib/utils";

export interface VoteButtonProps {
  /** Current total vote score for the thread. */
  voteScore: number;
  /** Whether the current user has already voted. */
  hasVoted: boolean;
  /** Weight of the user's vote (displayed in tooltip). */
  userWeight?: number;
  /** Called when the user toggles their vote. */
  onToggle: () => void;
  /** Whether the button is disabled (e.g. during API call). */
  disabled?: boolean;
}

/** Toggle vote button displaying current score and voted state. */
export function VoteButton({
  voteScore,
  hasVoted,
  userWeight = 1,
  onToggle,
  disabled = false,
}: VoteButtonProps): React.ReactNode {
  const [optimisticVoted, setOptimisticVoted] = useState(hasVoted);
  const [optimisticScore, setOptimisticScore] = useState(voteScore);

  const handleToggle = (): void => {
    if (disabled) return;
    const nextVoted = !optimisticVoted;
    setOptimisticVoted(nextVoted);
    setOptimisticScore(nextVoted ? optimisticScore + userWeight : optimisticScore - userWeight);
    onToggle();
  };

  return (
    <button
      onClick={handleToggle}
      disabled={disabled}
      data-testid="vote-button"
      title={`Vote weight: ${userWeight}`}
      className={cn(
        "inline-flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-sm font-medium transition-colors",
        optimisticVoted
          ? "border-primary bg-primary/10 text-primary"
          : "border-border text-muted-foreground hover:border-primary/50 hover:text-foreground",
        disabled && "cursor-not-allowed opacity-50",
      )}
    >
      <ThumbsUp
        className={cn("h-4 w-4", optimisticVoted && "fill-current")}
        data-testid="vote-icon"
      />
      <span data-testid="vote-score">{optimisticScore}</span>
    </button>
  );
}
