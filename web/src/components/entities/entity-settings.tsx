"use client";

import { Settings, Trash2 } from "lucide-react";
import { EntityForm, type EntityFormValues } from "./entity-form";
import type { EntityType } from "./entity-card";

export interface EntitySettingsProps {
  /** Entity type (org, space, board). */
  entityType: EntityType;
  /** Current entity values. */
  currentValues: EntityFormValues & { id: string; slug: string };
  /** Called when the user saves changes. */
  onSave: (values: EntityFormValues) => void;
  /** Called when the user requests deletion. */
  onDelete?: () => void;
  /** Whether a save operation is in progress. */
  loading?: boolean;
  /** Called to go back. */
  onCancel?: () => void;
}

/** Settings panel for managing an entity's details, metadata, and deletion. */
export function EntitySettings({
  entityType,
  currentValues,
  onSave,
  onDelete,
  loading = false,
  onCancel,
}: EntitySettingsProps): React.ReactNode {
  return (
    <div data-testid="entity-settings" className="flex flex-col gap-6">
      {/* Header */}
      <div className="flex items-center gap-2">
        <Settings className="h-5 w-5 text-muted-foreground" />
        <h1 className="text-xl font-bold text-foreground">{currentValues.slug} — Settings</h1>
      </div>

      {/* Edit form */}
      <div className="rounded-lg border border-border p-4">
        <EntityForm
          mode="edit"
          entityKind={entityType}
          initialValues={currentValues}
          onSubmit={onSave}
          onCancel={onCancel}
          loading={loading}
        />
      </div>

      {/* Danger zone */}
      {onDelete && (
        <div className="rounded-lg border border-destructive/30 p-4" data-testid="danger-zone">
          <h2 className="text-sm font-semibold text-destructive">Danger Zone</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Permanently delete this {entityType}. This action cannot be undone.
          </p>
          <button
            onClick={onDelete}
            disabled={loading}
            data-testid="entity-delete-btn"
            className="mt-3 inline-flex items-center gap-1 rounded-md bg-destructive px-3 py-1.5 text-sm font-medium text-destructive-foreground hover:bg-destructive/90 disabled:opacity-50"
          >
            <Trash2 className="h-4 w-4" />
            Delete {entityType}
          </button>
        </div>
      )}
    </div>
  );
}
