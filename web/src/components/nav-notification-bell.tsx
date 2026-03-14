"use client";

import { useAuth } from "@clerk/nextjs";
import { useEffect, useState } from "react";
import { NotificationBell } from "@/components/realtime/notification-bell";
import { useNotifications } from "@/hooks/use-notifications";

/** Wired notification bell for the top-level NavBar. Fetches Clerk token and delegates to useNotifications. */
export function NavNotificationBell(): React.ReactNode {
  const { getToken } = useAuth();
  const [token, setToken] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    getToken().then((t) => {
      if (active) setToken(t);
    });
    return () => {
      active = false;
    };
  }, [getToken]);

  const { notifications, unreadCount, loading, markRead, markAllRead } = useNotifications({
    token,
  });

  return (
    <NotificationBell
      notifications={notifications}
      unreadCount={unreadCount}
      loading={loading}
      onMarkRead={markRead}
      onMarkAllRead={markAllRead}
    />
  );
}
