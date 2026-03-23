"use client";

import { useState, type ReactNode } from "react";
import { useAuth } from "@clerk/nextjs";
import { ArrowBigUp } from "lucide-react";

import { clientMutate } from "@/lib/api-client";

interface VoteResult {
  voted: boolean;
  vote_score: number;
}

interface ForumVoteButtonProps {
  threadSlug: string;
  initialScore: number;
}

/** Inline vote button for forum thread cards. Stops link propagation. */
export function ForumVoteButton({ threadSlug, initialScore }: ForumVoteButtonProps): ReactNode {
  const { getToken, isSignedIn } = useAuth();
  const [score, setScore] = useState(initialScore);
  const [voted, setVoted] = useState(false);
  const [busy, setBusy] = useState(false);

  const handleVote = async (e: React.MouseEvent): Promise<void> => {
    e.preventDefault();
    e.stopPropagation();
    if (busy || !isSignedIn) return;
    setBusy(true);
    try {
      const token = await getToken();
      if (!token) return;
      const result = await clientMutate<VoteResult>(
        "POST",
        `/global-spaces/global-forum/threads/${encodeURIComponent(threadSlug)}/vote`,
        { token },
      );
      setScore(result.vote_score);
      setVoted(result.voted);
    } catch {
      // Best effort.
    } finally {
      setBusy(false);
    }
  };

  return (
    <button
      onClick={(e) => void handleVote(e)}
      data-testid="forum-vote-btn"
      disabled={busy}
      className="flex flex-col items-center gap-0.5 pt-0.5 rounded-lg px-1 py-1 transition-colors hover:bg-primary/10 disabled:opacity-50"
    >
      <ArrowBigUp
        className={`h-5 w-5 transition-colors ${voted ? "text-primary fill-primary" : "text-muted-foreground hover:text-primary"}`}
      />
      <span className={`text-sm font-semibold ${voted ? "text-primary" : "text-foreground"}`}>
        {score}
      </span>
    </button>
  );
}
