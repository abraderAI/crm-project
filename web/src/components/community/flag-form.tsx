"use client";

import { useState } from "react";
import { AlertTriangle } from "lucide-react";
import { cn } from "@/lib/utils";

export interface FlagFormProps {
  /** Called when the user submits a flag. */
  onSubmit: (reason: string) => void;
  /** Called when the user cancels. */
  onCancel?: () => void;
  /** Whether submission is in progress. */
  loading?: boolean;
}

const FLAG_REASONS = [
  "Spam or misleading",
  "Harassment or abuse",
  "Off-topic content",
  "Inappropriate language",
  "Other",
] as const;

/** Form for reporting/flagging content for moderation review. */
export function FlagForm({ onSubmit, onCancel, loading = false }: FlagFormProps): React.ReactNode {
  const [reason, setReason] = useState("");
  const [customReason, setCustomReason] = useState("");
  const [error, setError] = useState("");

  const handleSubmit = (e: React.FormEvent): void => {
    e.preventDefault();
    const finalReason = reason === "Other" ? customReason.trim() : reason;
    if (!finalReason) {
      setError("Please select or enter a reason.");
      return;
    }
    setError("");
    onSubmit(finalReason);
  };

  return (
    <form onSubmit={handleSubmit} data-testid="flag-form" className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <AlertTriangle className="h-5 w-5 text-destructive" data-testid="flag-icon" />
        <h3 className="text-sm font-semibold text-foreground">Report Content</h3>
      </div>

      <div className="flex flex-col gap-2" data-testid="flag-reasons">
        {FLAG_REASONS.map((r) => (
          <label
            key={r}
            className={cn(
              "flex items-center gap-2 rounded-md border px-3 py-2 text-sm transition-colors cursor-pointer",
              reason === r ? "border-primary bg-primary/5" : "border-border hover:bg-accent/50",
            )}
            data-testid={`flag-reason-${r.toLowerCase().replace(/\s+/g, "-")}`}
          >
            <input
              type="radio"
              name="flag-reason"
              value={r}
              checked={reason === r}
              onChange={() => setReason(r)}
              disabled={loading}
              className="accent-primary"
            />
            <span className="text-foreground">{r}</span>
          </label>
        ))}
      </div>

      {reason === "Other" && (
        <textarea
          value={customReason}
          onChange={(e) => setCustomReason(e.target.value)}
          placeholder="Describe the issue..."
          rows={3}
          disabled={loading}
          data-testid="flag-custom-reason"
          className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
        />
      )}

      {error && (
        <p className="text-xs text-destructive" data-testid="flag-error">
          {error}
        </p>
      )}

      <div className="flex items-center gap-2">
        <button
          type="submit"
          disabled={loading}
          data-testid="flag-submit-btn"
          className="rounded-md bg-destructive px-4 py-2 text-sm font-medium text-destructive-foreground hover:bg-destructive/90 disabled:opacity-50"
        >
          {loading ? "Submitting..." : "Submit Report"}
        </button>
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            disabled={loading}
            data-testid="flag-cancel-btn"
            className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-accent"
          >
            Cancel
          </button>
        )}
      </div>
    </form>
  );
}
