"use client";

import { useState } from "react";
import type { SpaceType } from "@/lib/api-types";
import { cn } from "@/lib/utils";

export type EntityFormMode = "create" | "edit";

export interface EntityFormValues {
  name: string;
  description: string;
  metadata: string;
  type?: SpaceType;
}

export interface EntityFormProps {
  /** Create or edit mode. */
  mode: EntityFormMode;
  /** Entity kind — determines whether the type selector is shown. */
  entityKind: "org" | "space" | "board";
  /** Initial values (pre-fill for edit). */
  initialValues?: Partial<EntityFormValues>;
  /** Called on valid form submission. */
  onSubmit: (values: EntityFormValues) => void;
  /** Called when the user cancels. */
  onCancel?: () => void;
  /** Disables the form during async operations. */
  loading?: boolean;
}

const SPACE_TYPES: SpaceType[] = ["general", "crm", "support", "community", "knowledge_base"];

/** Validate that metadata is valid JSON. */
function isValidJson(value: string): boolean {
  if (value.trim() === "") return true;
  try {
    JSON.parse(value);
    return true;
  } catch {
    return false;
  }
}

/** Reusable form for creating or editing an org, space, or board. */
export function EntityForm({
  mode,
  entityKind,
  initialValues,
  onSubmit,
  onCancel,
  loading = false,
}: EntityFormProps): React.ReactNode {
  const [name, setName] = useState(initialValues?.name ?? "");
  const [description, setDescription] = useState(initialValues?.description ?? "");
  const [metadata, setMetadata] = useState(initialValues?.metadata ?? "");
  const [type, setType] = useState<SpaceType>(initialValues?.type ?? "general");
  const [errors, setErrors] = useState<Record<string, string>>({});

  const validate = (): boolean => {
    const errs: Record<string, string> = {};
    if (!name.trim()) {
      errs["name"] = "Name is required";
    }
    if (metadata.trim() && !isValidJson(metadata)) {
      errs["metadata"] = "Metadata must be valid JSON";
    }
    setErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const handleSubmit = (e: React.FormEvent): void => {
    e.preventDefault();
    if (!validate()) return;
    onSubmit({
      name: name.trim(),
      description: description.trim(),
      metadata: metadata.trim() || "{}",
      ...(entityKind === "space" ? { type } : {}),
    });
  };

  const title = mode === "create" ? `Create ${entityKind}` : `Edit ${entityKind}`;

  return (
    <form onSubmit={handleSubmit} data-testid="entity-form" className="flex flex-col gap-4">
      <h2 className="text-lg font-semibold capitalize text-foreground">{title}</h2>

      {/* Name */}
      <div className="flex flex-col gap-1">
        <label htmlFor="entity-name" className="text-sm font-medium text-foreground">
          Name
        </label>
        <input
          id="entity-name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Enter name..."
          disabled={loading}
          data-testid="entity-name-input"
          className={cn(
            "rounded-md border bg-background px-3 py-2 text-sm text-foreground",
            errors["name"] ? "border-destructive" : "border-border",
            "focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary",
          )}
        />
        {errors["name"] && (
          <span className="text-xs text-destructive" data-testid="name-error">
            {errors["name"]}
          </span>
        )}
      </div>

      {/* Description */}
      <div className="flex flex-col gap-1">
        <label htmlFor="entity-description" className="text-sm font-medium text-foreground">
          Description
        </label>
        <textarea
          id="entity-description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Optional description..."
          rows={3}
          disabled={loading}
          data-testid="entity-description-input"
          className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
        />
      </div>

      {/* Space Type selector */}
      {entityKind === "space" && (
        <div className="flex flex-col gap-1">
          <label htmlFor="entity-type" className="text-sm font-medium text-foreground">
            Type
          </label>
          <select
            id="entity-type"
            value={type}
            onChange={(e) => setType(e.target.value as SpaceType)}
            disabled={loading}
            data-testid="entity-type-select"
            className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          >
            {SPACE_TYPES.map((t) => (
              <option key={t} value={t}>
                {t.replace("_", " ")}
              </option>
            ))}
          </select>
        </div>
      )}

      {/* Metadata JSON */}
      <div className="flex flex-col gap-1">
        <label htmlFor="entity-metadata" className="text-sm font-medium text-foreground">
          Metadata (JSON)
        </label>
        <textarea
          id="entity-metadata"
          value={metadata}
          onChange={(e) => setMetadata(e.target.value)}
          placeholder='{"key": "value"}'
          rows={3}
          disabled={loading}
          data-testid="entity-metadata-input"
          className={cn(
            "rounded-md border bg-background px-3 py-2 font-mono text-sm text-foreground",
            errors["metadata"] ? "border-destructive" : "border-border",
            "focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary",
          )}
        />
        {errors["metadata"] && (
          <span className="text-xs text-destructive" data-testid="metadata-error">
            {errors["metadata"]}
          </span>
        )}
      </div>

      {/* Actions */}
      <div className="flex items-center gap-2 pt-2">
        <button
          type="submit"
          disabled={loading}
          data-testid="entity-submit-btn"
          className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          {loading ? "Saving..." : mode === "create" ? "Create" : "Save"}
        </button>
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            disabled={loading}
            data-testid="entity-cancel-btn"
            className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-accent"
          >
            Cancel
          </button>
        )}
      </div>
    </form>
  );
}
