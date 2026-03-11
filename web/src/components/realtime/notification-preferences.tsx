"use client";

import { useCallback, useEffect, useState } from "react";
import { Save } from "lucide-react";
import type { DigestFrequency, NotificationChannel, NotificationType } from "@/lib/api-types";
import { cn } from "@/lib/utils";

/** A preference toggle for a specific notification type and channel. */
export interface PreferenceSetting {
  notificationType: NotificationType;
  channel: NotificationChannel;
  enabled: boolean;
}

/** Full preferences state. */
export interface NotificationPreferencesState {
  preferences: PreferenceSetting[];
  digestFrequency: DigestFrequency;
}

export interface NotificationPreferencesProps {
  /** Initial preferences state. */
  initialState: NotificationPreferencesState;
  /** Called when the user saves preferences. */
  onSave: (state: NotificationPreferencesState) => Promise<void>;
  /** Whether a save is in progress. */
  saving?: boolean;
}

/** Notification type labels. */
const TYPE_LABELS: Record<NotificationType, string> = {
  message: "New Messages",
  mention: "Mentions",
  stage_change: "Pipeline Stage Changes",
  assignment: "Assignments",
};

/** Channel labels. */
const CHANNEL_LABELS: Record<NotificationChannel, string> = {
  in_app: "In-App",
  email: "Email",
};

/** Digest frequency options. */
const DIGEST_OPTIONS: { value: DigestFrequency; label: string }[] = [
  { value: "none", label: "No digest" },
  { value: "daily", label: "Daily digest" },
  { value: "weekly", label: "Weekly digest" },
];

const NOTIFICATION_TYPES: NotificationType[] = ["message", "mention", "stage_change", "assignment"];
const CHANNELS: NotificationChannel[] = ["in_app", "email"];

/** Build default preferences (all enabled). */
export function buildDefaultPreferences(): PreferenceSetting[] {
  const prefs: PreferenceSetting[] = [];
  for (const notificationType of NOTIFICATION_TYPES) {
    for (const channel of CHANNELS) {
      prefs.push({ notificationType, channel, enabled: true });
    }
  }
  return prefs;
}

/** Notification preferences form. */
export function NotificationPreferences({
  initialState,
  onSave,
  saving = false,
}: NotificationPreferencesProps): React.ReactNode {
  const [preferences, setPreferences] = useState<PreferenceSetting[]>(initialState.preferences);
  const [digestFrequency, setDigestFrequency] = useState<DigestFrequency>(
    initialState.digestFrequency,
  );
  const [isDirty, setIsDirty] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);

  // Sync state from props when initialState changes.
  /* eslint-disable react-hooks/set-state-in-effect */
  useEffect(() => {
    setPreferences(initialState.preferences);
    setDigestFrequency(initialState.digestFrequency);
    setIsDirty(false);
  }, [initialState]);
  /* eslint-enable react-hooks/set-state-in-effect */

  const togglePreference = useCallback(
    (notificationType: NotificationType, channel: NotificationChannel) => {
      setPreferences((prev) =>
        prev.map((p) =>
          p.notificationType === notificationType && p.channel === channel
            ? { ...p, enabled: !p.enabled }
            : p,
        ),
      );
      setIsDirty(true);
      setSaveSuccess(false);
    },
    [],
  );

  const handleDigestChange = useCallback((frequency: DigestFrequency) => {
    setDigestFrequency(frequency);
    setIsDirty(true);
    setSaveSuccess(false);
  }, []);

  const handleSave = useCallback(async () => {
    await onSave({ preferences, digestFrequency });
    setIsDirty(false);
    setSaveSuccess(true);
  }, [onSave, preferences, digestFrequency]);

  const getPref = useCallback(
    (type: NotificationType, channel: NotificationChannel): boolean => {
      const found = preferences.find((p) => p.notificationType === type && p.channel === channel);
      return found?.enabled ?? true;
    },
    [preferences],
  );

  return (
    <div className="max-w-lg space-y-6" data-testid="notification-preferences">
      <div>
        <h2 className="text-lg font-semibold text-foreground">Notification Preferences</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Choose how you want to be notified for different events.
        </p>
      </div>

      {/* Per-type toggles */}
      <div className="space-y-4" data-testid="preference-toggles">
        {NOTIFICATION_TYPES.map((type) => (
          <div key={type} className="rounded-lg border border-border p-4">
            <h3 className="text-sm font-medium text-foreground" data-testid={`pref-type-${type}`}>
              {TYPE_LABELS[type]}
            </h3>
            <div className="mt-3 flex gap-4">
              {CHANNELS.map((channel) => {
                const enabled = getPref(type, channel);
                return (
                  <label
                    key={channel}
                    className="flex cursor-pointer items-center gap-2"
                    data-testid={`pref-toggle-${type}-${channel}`}
                  >
                    <button
                      type="button"
                      role="switch"
                      aria-checked={enabled}
                      onClick={() => togglePreference(type, channel)}
                      className={cn(
                        "relative inline-flex h-5 w-9 shrink-0 rounded-full transition-colors",
                        enabled ? "bg-primary" : "bg-muted",
                      )}
                      data-testid={`pref-switch-${type}-${channel}`}
                    >
                      <span
                        className={cn(
                          "pointer-events-none inline-block h-4 w-4 rounded-full bg-white shadow-sm transition-transform",
                          enabled ? "translate-x-4" : "translate-x-0.5",
                          "mt-0.5",
                        )}
                      />
                    </button>
                    <span className="text-sm text-muted-foreground">{CHANNEL_LABELS[channel]}</span>
                  </label>
                );
              })}
            </div>
          </div>
        ))}
      </div>

      {/* Digest frequency */}
      <div className="rounded-lg border border-border p-4" data-testid="digest-frequency-section">
        <h3 className="text-sm font-medium text-foreground">Email Digest</h3>
        <p className="mt-1 text-xs text-muted-foreground">
          Receive a summary of unread notifications on a schedule.
        </p>
        <div className="mt-3 flex gap-2" data-testid="digest-frequency-options">
          {DIGEST_OPTIONS.map((option) => (
            <button
              key={option.value}
              onClick={() => handleDigestChange(option.value)}
              className={cn(
                "rounded-md px-3 py-1.5 text-sm transition-colors",
                digestFrequency === option.value
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted text-muted-foreground hover:bg-muted/80",
              )}
              data-testid={`digest-option-${option.value}`}
            >
              {option.label}
            </button>
          ))}
        </div>
      </div>

      {/* Save button */}
      <div className="flex items-center gap-3">
        <button
          onClick={handleSave}
          disabled={!isDirty || saving}
          className={cn(
            "inline-flex items-center gap-2 rounded-md px-4 py-2 text-sm font-medium transition-colors",
            isDirty && !saving
              ? "bg-primary text-primary-foreground hover:bg-primary/90"
              : "cursor-not-allowed bg-muted text-muted-foreground",
          )}
          data-testid="save-preferences-btn"
        >
          <Save className="h-4 w-4" />
          {saving ? "Saving..." : "Save preferences"}
        </button>
        {saveSuccess && (
          <span className="text-sm text-green-600" data-testid="save-success-message">
            Preferences saved!
          </span>
        )}
      </div>
    </div>
  );
}
