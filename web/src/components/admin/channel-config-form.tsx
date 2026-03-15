"use client";

import { useState } from "react";
import { Save, RotateCcw } from "lucide-react";
import type { ChannelConfig, ChannelType } from "@/lib/api-types";

/** Field definition for a channel settings form. */
interface FieldDef {
  key: string;
  label: string;
  masked: boolean;
  type: "text" | "number";
}

/** Fields per channel type. */
const CHANNEL_FIELDS: Record<ChannelType, FieldDef[]> = {
  email: [
    { key: "imap_host", label: "IMAP Host", masked: false, type: "text" },
    { key: "imap_port", label: "IMAP Port", masked: false, type: "number" },
    { key: "imap_user", label: "IMAP User", masked: false, type: "text" },
    { key: "imap_password", label: "IMAP Password", masked: true, type: "text" },
    { key: "mailbox", label: "Mailbox", masked: false, type: "text" },
  ],
  voice: [
    { key: "livekit_url", label: "LiveKit URL", masked: false, type: "text" },
    { key: "livekit_api_key", label: "LiveKit API Key", masked: false, type: "text" },
    { key: "livekit_api_secret", label: "LiveKit API Secret", masked: true, type: "text" },
    { key: "webhook_token", label: "Webhook Token", masked: true, type: "text" },
  ],
  chat: [
    { key: "jwt_secret", label: "JWT Secret", masked: true, type: "text" },
    { key: "allowed_origins", label: "Allowed Origins", masked: false, type: "text" },
    { key: "max_session_minutes", label: "Max Session Minutes", masked: false, type: "number" },
  ],
};

/** Channel type display labels. */
const CHANNEL_LABELS: Record<ChannelType, string> = {
  email: "Email",
  voice: "Voice",
  chat: "Chat",
};

export interface ChannelConfigFormProps {
  /** Which channel type this form configures. */
  channelType: ChannelType;
  /** Initial configuration data. */
  initialConfig: ChannelConfig | null;
  /** Called when the form is saved. */
  onSave: (settings: Record<string, string>, enabled: boolean) => Promise<void> | void;
}

/** Parse settings JSON string into a record. */
function parseSettings(raw: string | undefined): Record<string, string> {
  if (!raw) return {};
  try {
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed === "object" && parsed !== null && !Array.isArray(parsed)) {
      const result: Record<string, string> = {};
      for (const [k, v] of Object.entries(parsed as Record<string, unknown>)) {
        result[k] = String(v ?? "");
      }
      return result;
    }
    return {};
  } catch {
    return {};
  }
}

/** Config form for a specific channel type with dynamic fields and masked secrets. */
export function ChannelConfigForm({
  channelType,
  initialConfig,
  onSave,
}: ChannelConfigFormProps): React.ReactNode {
  const fields = CHANNEL_FIELDS[channelType];
  const initialSettings = parseSettings(initialConfig?.settings);
  const [values, setValues] = useState<Record<string, string>>(() => {
    const v: Record<string, string> = {};
    for (const f of fields) {
      v[f.key] = f.masked ? "" : (initialSettings[f.key] ?? "");
    }
    return v;
  });
  const [enabled, setEnabled] = useState(initialConfig?.enabled ?? false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const handleChange = (key: string, value: string): void => {
    setValues((prev) => ({ ...prev, [key]: value }));
  };

  const handleReset = (): void => {
    const v: Record<string, string> = {};
    for (const f of fields) {
      v[f.key] = f.masked ? "" : (initialSettings[f.key] ?? "");
    }
    setValues(v);
    setEnabled(initialConfig?.enabled ?? false);
    setError("");
  };

  const handleSubmit = async (e: React.FormEvent): Promise<void> => {
    e.preventDefault();
    setError("");
    setSaving(true);
    try {
      // Build settings: include non-masked fields always, masked fields only if non-empty.
      const settings: Record<string, string> = {};
      for (const f of fields) {
        if (f.masked) {
          const val = values[f.key];
          const existing = initialSettings[f.key];
          if (val?.trim()) {
            settings[f.key] = val.trim();
          } else if (existing) {
            // Keep existing secret value from initial config.
            settings[f.key] = existing;
          }
        } else {
          settings[f.key] = values[f.key] ?? "";
        }
      }
      await onSave(settings, enabled);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save configuration.");
    } finally {
      setSaving(false);
    }
  };

  return (
    <form
      onSubmit={handleSubmit}
      data-testid="channel-config-form"
      className="rounded-lg border border-border p-6"
    >
      <h2 className="text-lg font-semibold text-foreground">
        {CHANNEL_LABELS[channelType]} Configuration
      </h2>

      <div className="mt-4 flex flex-col gap-4">
        {fields.map((field) => (
          <div key={field.key} className="flex flex-col gap-1">
            <label htmlFor={`field-${field.key}`} className="text-sm font-medium text-foreground">
              {field.label}
            </label>
            {field.masked && initialSettings[field.key] && (
              <p
                className="text-xs text-muted-foreground"
                data-testid={`field-masked-${field.key}`}
              >
                Current value: ••••••••
              </p>
            )}
            <input
              id={`field-${field.key}`}
              type={field.masked ? "password" : field.type === "number" ? "number" : "text"}
              value={values[field.key] ?? ""}
              onChange={(e) => handleChange(field.key, e.target.value)}
              placeholder={field.masked ? "Enter new value to update" : ""}
              data-testid={`field-input-${field.key}`}
              className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
            />
          </div>
        ))}

        <div className="flex items-center gap-3">
          <label htmlFor="channel-enabled" className="text-sm font-medium text-foreground">
            Enabled
          </label>
          <button
            id="channel-enabled"
            type="button"
            role="switch"
            aria-checked={enabled}
            onClick={() => setEnabled(!enabled)}
            data-testid="channel-enabled-toggle"
            className={`relative h-6 w-11 rounded-full transition-colors ${
              enabled ? "bg-primary" : "bg-muted"
            }`}
          >
            <span
              className={`absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white transition-transform ${
                enabled ? "translate-x-5" : "translate-x-0"
              }`}
            />
          </button>
        </div>

        {error && (
          <p className="text-xs text-destructive" data-testid="config-error">
            {error}
          </p>
        )}

        <div className="flex gap-2">
          <button
            type="submit"
            disabled={saving}
            data-testid="config-save-btn"
            className="inline-flex items-center gap-1 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            <Save className="h-4 w-4" />
            {saving ? "Saving..." : "Save"}
          </button>
          <button
            type="button"
            onClick={handleReset}
            disabled={saving}
            data-testid="config-reset-btn"
            className="inline-flex items-center gap-1 rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-accent disabled:opacity-50"
          >
            <RotateCcw className="h-4 w-4" />
            Reset
          </button>
        </div>
      </div>
    </form>
  );
}
