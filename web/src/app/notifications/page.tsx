import Link from "next/link";
import { Settings } from "lucide-react";

import { fetchNotifications } from "@/lib/user-api";
import { NotificationFeed } from "@/components/realtime/notification-feed";

/** Notifications page — displays the current user's notification feed. */
export default async function NotificationsPage(): Promise<React.ReactNode> {
  const { data: notifications } = await fetchNotifications();

  return (
    <div className="mx-auto max-w-2xl space-y-6 p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold text-foreground">Notifications</h1>
        <Link
          href="/notifications/preferences"
          className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
          data-testid="notification-preferences-link"
        >
          <Settings className="h-4 w-4" />
          Preferences
        </Link>
      </div>
      <NotificationFeed notifications={notifications} />
    </div>
  );
}
