import { fetchNotifications } from "@/lib/user-api";
import { NotificationFeed } from "@/components/realtime/notification-feed";

/** Notifications page — displays the current user's notification feed. */
export default async function NotificationsPage(): Promise<React.ReactNode> {
  const { data: notifications } = await fetchNotifications();

  return (
    <div className="mx-auto max-w-2xl space-y-6 p-6">
      <h1 className="text-xl font-bold text-foreground">Notifications</h1>
      <NotificationFeed notifications={notifications} />
    </div>
  );
}
