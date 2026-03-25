"use client";

import { useState } from "react";
import type { PipelineStage } from "@/lib/crm-types";
import { STAGE_LABELS, PIPELINE_STAGES } from "@/lib/crm-types";

export interface StageTransitionModalProps {
  /** Current stage of the opportunity. */
  currentStage: PipelineStage;
  /** Target stage to transition to. */
  targetStage: PipelineStage;
  /** Called when the user confirms the transition. */
  onConfirm: (data: {
    stage: string;
    reason?: string;
    close_reason?: string;
    comment?: string;
  }) => void;
  /** Called when the user cancels. */
  onCancel: () => void;
  /** Whether the transition is in progress. */
  isLoading?: boolean;
}

/** Determine if a stage transition is backward in the pipeline. */
export function isBackwardMove(from: PipelineStage, to: PipelineStage): boolean {
  const fromIdx = PIPELINE_STAGES.indexOf(from);
  const toIdx = PIPELINE_STAGES.indexOf(to);
  return toIdx < fromIdx;
}

/** Determine if the target stage is a close stage. */
export function isCloseStage(stage: PipelineStage): boolean {
  return stage === "closed_won" || stage === "closed_lost";
}

/** Modal for stage transitions requiring reason (backward moves) or close reason (won/lost). */
export function StageTransitionModal({
  currentStage,
  targetStage,
  onConfirm,
  onCancel,
  isLoading = false,
}: StageTransitionModalProps): React.ReactNode {
  const [reason, setReason] = useState("");
  const [comment, setComment] = useState("");

  const needsReason = isBackwardMove(currentStage, targetStage);
  const needsCloseReason = isCloseStage(targetStage);
  const title = needsCloseReason
    ? `Close as ${STAGE_LABELS[targetStage]}`
    : `Move to ${STAGE_LABELS[targetStage]}`;

  const handleSubmit = (e: React.FormEvent): void => {
    e.preventDefault();
    const data: { stage: string; reason?: string; close_reason?: string; comment?: string } = {
      stage: targetStage,
    };
    if (needsReason && reason.trim()) data.reason = reason.trim();
    if (needsCloseReason && reason.trim()) data.close_reason = reason.trim();
    if (comment.trim()) data.comment = comment.trim();
    onConfirm(data);
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      data-testid="stage-transition-modal"
      onClick={onCancel}
    >
      <div
        className="w-full max-w-md rounded-lg border border-border bg-background p-6 shadow-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-lg font-semibold text-foreground" data-testid="transition-modal-title">
          {title}
        </h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Moving from <strong>{STAGE_LABELS[currentStage]}</strong> to{" "}
          <strong>{STAGE_LABELS[targetStage]}</strong>
        </p>

        <form onSubmit={handleSubmit} className="mt-4 space-y-3">
          {(needsReason || needsCloseReason) && (
            <div>
              <label
                htmlFor="transition-reason"
                className="mb-1 block text-sm font-medium text-foreground"
              >
                {needsCloseReason ? "Close Reason *" : "Reason for moving backward *"}
              </label>
              <textarea
                id="transition-reason"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                required
                rows={3}
                data-testid="transition-reason-input"
                className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                placeholder={
                  needsCloseReason
                    ? "Why is this deal being closed?"
                    : "Why is this moving backward?"
                }
              />
            </div>
          )}

          <div>
            <label
              htmlFor="transition-comment"
              className="mb-1 block text-sm font-medium text-foreground"
            >
              Comment (optional)
            </label>
            <textarea
              id="transition-comment"
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              rows={2}
              data-testid="transition-comment-input"
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onCancel}
              className="rounded-md border border-border px-4 py-2 text-sm text-foreground hover:bg-accent"
              data-testid="transition-cancel-btn"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isLoading || ((needsReason || needsCloseReason) && !reason.trim())}
              className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              data-testid="transition-confirm-btn"
            >
              {isLoading ? "Saving..." : "Confirm"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
