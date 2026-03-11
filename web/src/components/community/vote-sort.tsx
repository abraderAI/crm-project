"use client";

import { ArrowDownUp } from "lucide-react";
import { cn } from "@/lib/utils";
import type { ThreadSortOption } from "@/lib/api-types";

export interface VoteSortProps {
  /** The currently active sort option. */
  value: ThreadSortOption;
  /** Called when the user selects a different sort option. */
  onChange: (option: ThreadSortOption) => void;
}

const SORT_OPTIONS: { value: ThreadSortOption; label: string }[] = [
  { value: "votes", label: "Top voted" },
  { value: "newest", label: "Newest" },
  { value: "oldest", label: "Oldest" },
];

/** Sort control for community thread lists — sort by votes, newest, or oldest. */
export function VoteSort({ value, onChange }: VoteSortProps): React.ReactNode {
  return (
    <div className="flex items-center gap-2" data-testid="vote-sort">
      <ArrowDownUp className="h-4 w-4 text-muted-foreground" data-testid="vote-sort-icon" />
      <div className="flex rounded-md border border-border" data-testid="vote-sort-options">
        {SORT_OPTIONS.map((option) => (
          <button
            key={option.value}
            onClick={() => onChange(option.value)}
            data-testid={`sort-option-${option.value}`}
            className={cn(
              "px-3 py-1.5 text-xs font-medium transition-colors first:rounded-l-md last:rounded-r-md",
              value === option.value
                ? "bg-primary text-primary-foreground"
                : "text-muted-foreground hover:bg-accent hover:text-foreground",
            )}
          >
            {option.label}
          </button>
        ))}
      </div>
    </div>
  );
}
