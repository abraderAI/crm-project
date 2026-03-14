"use client";

import { useAuth } from "@clerk/nextjs";
import { useCallback, useState } from "react";
import {
  NotificationPreferences,
  type NotificationPreferencesState,
} from "./notification-preferences";
import { saveNotificationPreferences, saveDigestSchedule } from "@/lib/entity-api";

export interface NotificationPreferencesViewProps {
  /** Initial preferences state fetched on the server. */
  initialState: NotificationPreferencesState;
}

/** Client wrapper that wires NotificationPreferences to the save APIs via Clerk auth. */
export function NotificationPreferencesView({
  initialState,
}: NotificationPreferencesViewProps): React.ReactNode {
  const { getToken } = useAuth();
  const [saving, setSaving] = useState(false);

  const handleSave = useCallback(
    async (state: NotificationPreferencesState) => {
      const token = await getToken();
      if (!token) return;
      setSaving(true);
      try {
        await Promise.all([
          saveNotificationPreferences(token, state.preferences),
          saveDigestSchedule(token, state.digestFrequency),
        ]);
      } finally {
        setSaving(false);
      }
    },
    [getToken],
  );

  return (
    <NotificationPreferences initialState={initialState} onSave={handleSave} saving={saving} />
  );
}
