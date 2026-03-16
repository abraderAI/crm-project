"use client";

import type { ReactNode } from "react";

export interface ResetConfirmDialogProps {
  /** Whether the dialog is visible. */
  open: boolean;
  /** Called when the user confirms the reset. */
  onConfirm: () => void;
  /** Called when the user cancels. */
  onCancel: () => void;
}

/** Confirmation dialog shown before resetting home screen layout to defaults. */
export function ResetConfirmDialog({
  open,
  onConfirm,
  onCancel,
}: ResetConfirmDialogProps): ReactNode {
  if (!open) {
    return null;
  }

  return (
    <div
      data-testid="reset-confirm-dialog"
      role="dialog"
      aria-modal="true"
      aria-label="Confirm reset"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
    >
      <div className="mx-4 w-full max-w-sm rounded-lg border border-border bg-background p-6 shadow-lg">
        <h4 className="text-sm font-semibold text-foreground" data-testid="reset-dialog-title">
          Reset to defaults?
        </h4>
        <p className="mt-2 text-sm text-muted-foreground" data-testid="reset-dialog-description">
          This will remove all your customizations and restore the default home screen layout for
          your tier. This action cannot be undone.
        </p>
        <div className="mt-4 flex justify-end gap-2">
          <button
            data-testid="reset-dialog-cancel"
            onClick={onCancel}
            className="rounded-md border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:bg-accent/50"
          >
            Cancel
          </button>
          <button
            data-testid="reset-dialog-confirm"
            onClick={onConfirm}
            className="rounded-md bg-red-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-600"
          >
            Reset
          </button>
        </div>
      </div>
    </div>
  );
}
