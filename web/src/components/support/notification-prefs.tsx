"use client";

import { useState, type ReactNode } from "react";
import { useAuth } from "@clerk/nextjs";
import { Bell, BellOff } from "lucide-react";

import { setTicketNotificationPref } from "@/lib/support-api";

/** Props for NotificationPrefs. */
export interface NotificationPrefsProps {
  /** Slug of the ticket whose preferences are being configured. */
  ticketSlug: string;
  /** Current detail level read from ticket metadata. */
  currentLevel: "full" | "privacy";
}

/**
 * NotificationPrefs renders a toggle that lets the ticket owner choose whether
 * notification emails include the agent reply body ("full") or are privacy-mode
 * link-only emails ("privacy").
 */
export function NotificationPrefs({ ticketSlug, currentLevel }: NotificationPrefsProps): ReactNode {
  const { getToken } = useAuth();
  const [level, setLevel] = useState<"full" | "privacy">(currentLevel);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const handleChange = async (next: "full" | "privacy"): Promise<void> => {
    if (next === level) return;
    setError("");
    setSaving(true);
    try {
      const token = await getToken();
      if (!token) return;
      await setTicketNotificationPref(token, ticketSlug, next);
      setLevel(next);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save preference");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div
      data-testid="notification-prefs"
      className="rounded-lg border border-border bg-background p-4"
    >
      <h4 className="mb-3 text-sm font-semibold text-foreground">Email notifications</h4>
      <div className="flex flex-col gap-2">
        {/* Full detail option */}
        <button
          data-testid="notif-pref-full"
          onClick={() => void handleChange("full")}
          disabled={saving}
          className={`flex items-start gap-3 rounded-md border p-3 text-left transition-colors ${
            level === "full" ? "border-primary bg-primary/5" : "border-border hover:bg-accent"
          }`}
        >
          <Bell
            className={`mt-0.5 h-4 w-4 shrink-0 ${level === "full" ? "text-primary" : "text-muted-foreground"}`}
          />
          <div>
            <p className="text-xs font-medium text-foreground">Full detail</p>
            <p className="text-xs text-muted-foreground">
              Include the agent&apos;s reply in the notification email.
            </p>
          </div>
        </button>

        {/* Privacy option */}
        <button
          data-testid="notif-pref-privacy"
          onClick={() => void handleChange("privacy")}
          disabled={saving}
          className={`flex items-start gap-3 rounded-md border p-3 text-left transition-colors ${
            level === "privacy" ? "border-primary bg-primary/5" : "border-border hover:bg-accent"
          }`}
        >
          <BellOff
            className={`mt-0.5 h-4 w-4 shrink-0 ${level === "privacy" ? "text-primary" : "text-muted-foreground"}`}
          />
          <div>
            <p className="text-xs font-medium text-foreground">Privacy mode</p>
            <p className="text-xs text-muted-foreground">
              Only send a link to the ticket — no reply content in the email.
            </p>
          </div>
        </button>
      </div>

      {saving && (
        <p data-testid="notif-prefs-saving" className="mt-2 text-xs text-muted-foreground">
          Saving…
        </p>
      )}
      {error && (
        <p data-testid="notif-prefs-error" className="mt-2 text-xs text-red-600">
          {error}
        </p>
      )}
    </div>
  );
}
