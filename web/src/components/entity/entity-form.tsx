"use client";

import { useState } from "react";
import type { SpaceType } from "@/lib/api-types";
import {
  MetadataEditor,
  entriesToRecord,
  recordToEntries,
  type MetadataEntry,
} from "./metadata-editor";

export interface EntityFormValues {
  name: string;
  description: string;
  type?: SpaceType;
  metadata: Record<string, string>;
}

export interface EntityFormProps {
  /** Entity type label (e.g. "Organization", "Space", "Board"). */
  entityLabel: string;
  /** Whether to show the space type selector. */
  showTypeSelector?: boolean;
  /** Initial values for edit mode. */
  initialValues?: Partial<EntityFormValues>;
  /** Called on form submission. */
  onSubmit: (values: EntityFormValues) => void;
  /** Called when cancel is clicked. */
  onCancel?: () => void;
  /** Whether the form is submitting. */
  submitting?: boolean;
  /** Submit button label override. */
  submitLabel?: string;
}

const SPACE_TYPES: SpaceType[] = ["general", "crm", "support", "community", "knowledge_base"];

/** Form for creating or editing an entity (Org, Space, or Board). */
export function EntityForm({
  entityLabel,
  showTypeSelector = false,
  initialValues,
  onSubmit,
  onCancel,
  submitting = false,
  submitLabel,
}: EntityFormProps): React.ReactNode {
  const [name, setName] = useState(initialValues?.name ?? "");
  const [description, setDescription] = useState(initialValues?.description ?? "");
  const [spaceType, setSpaceType] = useState<SpaceType>(initialValues?.type ?? "general");
  const [metadataEntries, setMetadataEntries] = useState<MetadataEntry[]>(() =>
    recordToEntries(initialValues?.metadata ?? {}),
  );
  const [errors, setErrors] = useState<Record<string, string>>({});

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};
    if (!name.trim()) {
      newErrors["name"] = "Name is required";
    }
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent): void => {
    e.preventDefault();
    if (!validate()) return;

    onSubmit({
      name: name.trim(),
      description: description.trim(),
      type: showTypeSelector ? spaceType : undefined,
      metadata: entriesToRecord(metadataEntries),
    });
  };

  const isEdit = !!initialValues?.name;
  const buttonText = submitLabel ?? (isEdit ? `Update ${entityLabel}` : `Create ${entityLabel}`);

  return (
    <form onSubmit={handleSubmit} data-testid="entity-form" className="space-y-4">
      {/* Name */}
      <div>
        <label htmlFor="entity-name" className="mb-1 block text-sm font-medium text-foreground">
          Name
        </label>
        <input
          id="entity-name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={`${entityLabel} name`}
          disabled={submitting}
          data-testid="entity-name-input"
          className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
        />
        {errors["name"] && (
          <p className="mt-1 text-xs text-destructive" data-testid="entity-name-error">
            {errors["name"]}
          </p>
        )}
      </div>

      {/* Description */}
      <div>
        <label htmlFor="entity-desc" className="mb-1 block text-sm font-medium text-foreground">
          Description
        </label>
        <textarea
          id="entity-desc"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Optional description"
          disabled={submitting}
          rows={3}
          data-testid="entity-desc-input"
          className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
        />
      </div>

      {/* Space type */}
      {showTypeSelector && (
        <div>
          <label htmlFor="entity-type" className="mb-1 block text-sm font-medium text-foreground">
            Type
          </label>
          <select
            id="entity-type"
            value={spaceType}
            onChange={(e) => setSpaceType(e.target.value as SpaceType)}
            disabled={submitting}
            data-testid="entity-type-select"
            className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          >
            {SPACE_TYPES.map((t) => (
              <option key={t} value={t}>
                {t.replace("_", " ")}
              </option>
            ))}
          </select>
        </div>
      )}

      {/* Metadata */}
      <MetadataEditor
        entries={metadataEntries}
        onChange={setMetadataEntries}
        disabled={submitting}
      />

      {/* Actions */}
      <div className="flex items-center gap-3 pt-2">
        <button
          type="submit"
          disabled={submitting}
          data-testid="entity-submit-btn"
          className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
        >
          {submitting ? "Saving..." : buttonText}
        </button>
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            disabled={submitting}
            data-testid="entity-cancel-btn"
            className="rounded-md border border-border px-4 py-2 text-sm text-foreground transition-colors hover:bg-accent disabled:opacity-50"
          >
            Cancel
          </button>
        )}
      </div>
    </form>
  );
}
