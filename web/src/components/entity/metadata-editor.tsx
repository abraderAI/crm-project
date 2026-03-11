"use client";

import { Plus, Trash2 } from "lucide-react";
import { useCallback, useState } from "react";

export interface MetadataEntry {
  key: string;
  value: string;
}

export interface MetadataEditorProps {
  /** Initial entries. */
  entries: MetadataEntry[];
  /** Called whenever entries change. */
  onChange: (entries: MetadataEntry[]) => void;
  /** Whether the editor is disabled. */
  disabled?: boolean;
}

/** Convert a flat Record to entry array. */
export function recordToEntries(record: Record<string, unknown>): MetadataEntry[] {
  return Object.entries(record).map(([key, value]) => ({
    key,
    value: typeof value === "string" ? value : JSON.stringify(value),
  }));
}

/** Convert entry array back to Record. */
export function entriesToRecord(entries: MetadataEntry[]): Record<string, string> {
  const result: Record<string, string> = {};
  for (const entry of entries) {
    if (entry.key.trim()) {
      result[entry.key.trim()] = entry.value;
    }
  }
  return result;
}

/** Key-value metadata editor for adding, editing, and removing JSON metadata pairs. */
export function MetadataEditor({
  entries,
  onChange,
  disabled = false,
}: MetadataEditorProps): React.ReactNode {
  const [localEntries, setLocalEntries] = useState<MetadataEntry[]>(entries);

  const updateEntries = useCallback(
    (updated: MetadataEntry[]) => {
      setLocalEntries(updated);
      onChange(updated);
    },
    [onChange],
  );

  const addEntry = (): void => {
    updateEntries([...localEntries, { key: "", value: "" }]);
  };

  const removeEntry = (index: number): void => {
    updateEntries(localEntries.filter((_, i) => i !== index));
  };

  const updateKey = (index: number, key: string): void => {
    const updated = localEntries.map((entry, i) => (i === index ? { ...entry, key } : entry));
    updateEntries(updated);
  };

  const updateValue = (index: number, value: string): void => {
    const updated = localEntries.map((entry, i) => (i === index ? { ...entry, value } : entry));
    updateEntries(updated);
  };

  return (
    <div data-testid="metadata-editor">
      <div className="mb-2 flex items-center justify-between">
        <label className="text-sm font-medium text-foreground">Metadata</label>
        <button
          type="button"
          onClick={addEntry}
          disabled={disabled}
          data-testid="metadata-add-btn"
          className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs text-primary hover:bg-accent disabled:opacity-50"
        >
          <Plus className="h-3 w-3" />
          Add field
        </button>
      </div>

      {localEntries.length === 0 ? (
        <p className="text-xs text-muted-foreground" data-testid="metadata-empty">
          No metadata fields. Click &quot;Add field&quot; to add one.
        </p>
      ) : (
        <div className="space-y-2">
          {localEntries.map((entry, index) => (
            <div
              key={index}
              className="flex items-center gap-2"
              data-testid={`metadata-row-${index}`}
            >
              <input
                type="text"
                value={entry.key}
                onChange={(e) => updateKey(index, e.target.value)}
                placeholder="Key"
                disabled={disabled}
                data-testid={`metadata-key-${index}`}
                className="h-8 flex-1 rounded-md border border-border bg-background px-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <input
                type="text"
                value={entry.value}
                onChange={(e) => updateValue(index, e.target.value)}
                placeholder="Value"
                disabled={disabled}
                data-testid={`metadata-value-${index}`}
                className="h-8 flex-1 rounded-md border border-border bg-background px-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <button
                type="button"
                onClick={() => removeEntry(index)}
                disabled={disabled}
                data-testid={`metadata-remove-${index}`}
                aria-label={`Remove field ${index}`}
                className="rounded-md p-1 text-muted-foreground hover:bg-destructive/10 hover:text-destructive disabled:opacity-50"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
