import Link from "next/link";
import { ChevronLeft } from "lucide-react";

import { fetchNotificationPreferences, fetchDigestSchedule } from "@/lib/user-api";
import { buildDefaultPreferences } from "@/components/realtime/notification-preferences";
import { NotificationPreferencesView } from "@/components/realtime/notification-preferences-view";
import type { PreferenceSetting } from "@/components/realtime/notification-preferences";

/** Notification preferences page — manage per-type/channel toggles and digest schedule. */
export default async function NotificationPreferencesPage(): Promise<React.ReactNode> {
  const [{ data: apiPrefs }, digest] = await Promise.all([
    fetchNotificationPreferences(),
    fetchDigestSchedule(),
  ]);

  // Map API preferences to component state, falling back to defaults for any missing combos.
  const defaults = buildDefaultPreferences();
  const preferences: PreferenceSetting[] = defaults.map((d) => {
    const match = apiPrefs.find(
      (p) => p.notification_type === d.notificationType && p.channel === d.channel,
    );
    return match ? { ...d, enabled: match.enabled } : d;
  });

  return (
    <div className="mx-auto max-w-2xl space-y-6 p-6">
      <div className="flex items-center gap-2">
        <Link
          href="/notifications"
          className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
          data-testid="back-to-notifications"
        >
          <ChevronLeft className="h-4 w-4" />
          Back to Notifications
        </Link>
      </div>

      <NotificationPreferencesView
        initialState={{ preferences, digestFrequency: digest.frequency }}
      />
    </div>
  );
}
