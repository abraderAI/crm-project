"use client";

import { useEffect, useState } from "react";
import { Save } from "lucide-react";
import { useAuth } from "@clerk/nextjs";
import { clientMutate } from "@/lib/api-client";

/** Infer the setting value type for rendering the correct input. */
type SettingType = "string" | "number" | "boolean" | "json";

/** Determine the input type for a given value. */
function inferType(value: unknown): SettingType {
  if (typeof value === "boolean") return "boolean";
  if (typeof value === "number") return "number";
  if (typeof value === "object" && value !== null) return "json";
  return "string";
}

/** Internal representation of a setting for editing. */
interface SettingEntry {
  key: string;
  type: SettingType;
  /** Serialized value: JSON string for objects/arrays, raw string otherwise. */
  raw: string;
  /** Boolean value for toggles. */
  boolValue: boolean;
  /** Number value for number inputs. */
  numValue: number;
}

/** Build initial setting entries from the settings map. */
function buildEntries(settings: Record<string, unknown>): SettingEntry[] {
  return Object.entries(settings).map(([key, value]) => {
    const type = inferType(value);
    return {
      key,
      type,
      raw: type === "json" ? JSON.stringify(value, null, 2) : String(value ?? ""),
      boolValue: type === "boolean" ? (value as boolean) : false,
      numValue: type === "number" ? (value as number) : 0,
    };
  });
}

/** Serialize an entry back to its typed value for the PATCH body. */
function serializeEntry(entry: SettingEntry): unknown {
  switch (entry.type) {
    case "boolean":
      return entry.boolValue;
    case "number":
      return entry.numValue;
    case "json":
      try {
        return JSON.parse(entry.raw) as unknown;
      } catch {
        return entry.raw;
      }
    default:
      return entry.raw;
  }
}

export interface SystemSettingsProps {
  /** Initial settings loaded server-side from GET /v1/admin/settings. */
  initialSettings: Record<string, unknown>;
}

/** Editable key-value settings editor for platform-wide configuration. */
export function SystemSettings({ initialSettings }: SystemSettingsProps): React.ReactNode {
  const { getToken } = useAuth();
  const [entries, setEntries] = useState<SettingEntry[]>(() => buildEntries(initialSettings));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [toast, setToast] = useState("");

  useEffect(() => {
    if (!toast) return;
    const timer = setTimeout(() => setToast(""), 3000);
    return () => clearTimeout(timer);
  }, [toast]);

  const updateEntry = (key: string, patch: Partial<SettingEntry>): void => {
    setEntries((prev) => prev.map((e) => (e.key === key ? { ...e, ...patch } : e)));
  };

  const handleSave = async (): Promise<void> => {
    setError("");
    setSaving(true);
    try {
      const token = await getToken();
      const body: Record<string, unknown> = {};
      for (const entry of entries) {
        body[entry.key] = serializeEntry(entry);
      }
      await clientMutate<Record<string, unknown>>("PATCH", "/admin/settings", {
        token,
        body,
      });
      setToast("Settings saved");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save settings.");
    } finally {
      setSaving(false);
    }
  };

  const isEmpty = entries.length === 0;

  return (
    <div data-testid="system-settings-form" className="flex flex-col gap-6">
      <h2 className="text-lg font-semibold text-foreground">System Settings</h2>

      {isEmpty && (
        <p data-testid="settings-empty" className="text-sm text-muted-foreground">
          No settings configured yet.
        </p>
      )}

      {!isEmpty && (
        <div className="flex flex-col gap-4">
          {entries.map((entry) => (
            <div
              key={entry.key}
              data-testid={`setting-row-${entry.key}`}
              className="flex flex-col gap-1 rounded-lg border border-border p-4"
            >
              <label
                htmlFor={`setting-${entry.key}`}
                className="text-sm font-medium text-foreground"
              >
                {entry.key}
              </label>

              {entry.type === "boolean" && (
                <button
                  id={`setting-${entry.key}`}
                  type="button"
                  role="switch"
                  aria-checked={entry.boolValue}
                  onClick={() => updateEntry(entry.key, { boolValue: !entry.boolValue })}
                  data-testid={`setting-input-${entry.key}`}
                  className={`relative h-6 w-11 rounded-full transition-colors ${
                    entry.boolValue ? "bg-primary" : "bg-muted"
                  }`}
                >
                  <span
                    className={`absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white transition-transform ${
                      entry.boolValue ? "translate-x-5" : "translate-x-0"
                    }`}
                  />
                </button>
              )}

              {entry.type === "number" && (
                <input
                  id={`setting-${entry.key}`}
                  type="number"
                  value={entry.numValue}
                  onChange={(e) =>
                    updateEntry(entry.key, { numValue: Number(e.target.value) || 0 })
                  }
                  data-testid={`setting-input-${entry.key}`}
                  className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
                />
              )}

              {entry.type === "string" && (
                <input
                  id={`setting-${entry.key}`}
                  type="text"
                  value={entry.raw}
                  onChange={(e) => updateEntry(entry.key, { raw: e.target.value })}
                  data-testid={`setting-input-${entry.key}`}
                  className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
                />
              )}

              {entry.type === "json" && (
                <textarea
                  id={`setting-${entry.key}`}
                  value={entry.raw}
                  onChange={(e) => updateEntry(entry.key, { raw: e.target.value })}
                  rows={Math.min(Math.max(entry.raw.split("\n").length, 3), 12)}
                  data-testid={`setting-input-${entry.key}`}
                  className="rounded-md border border-border bg-background px-3 py-2 font-mono text-sm text-foreground"
                />
              )}
            </div>
          ))}
        </div>
      )}

      {error && (
        <p className="text-xs text-destructive" data-testid="settings-error">
          {error}
        </p>
      )}

      {toast && (
        <p
          className="rounded-md bg-green-100 px-3 py-2 text-sm text-green-800 dark:bg-green-900 dark:text-green-200"
          data-testid="settings-toast"
        >
          {toast}
        </p>
      )}

      <div className="flex gap-2">
        <button
          type="button"
          onClick={handleSave}
          disabled={saving}
          data-testid="settings-save-btn"
          className="inline-flex items-center gap-1 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          <Save className="h-4 w-4" />
          {saving ? "Saving..." : "Save"}
        </button>
      </div>
    </div>
  );
}
