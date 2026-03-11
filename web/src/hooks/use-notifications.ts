"use client";

import { useCallback, useEffect, useState } from "react";
import type { Notification, PaginatedResponse, WSMessage } from "@/lib/api-types";
import { clientMutate, parseResponse, buildUrl, buildHeaders } from "@/lib/api-client";

/** Options for the useNotifications hook. */
export interface UseNotificationsOptions {
  /** JWT token. */
  token: string | null;
  /** Whether to enable fetching. */
  enabled?: boolean;
}

/** Return value of the useNotifications hook. */
export interface UseNotificationsReturn {
  /** Current notification list (newest first). */
  notifications: Notification[];
  /** Number of unread notifications. */
  unreadCount: number;
  /** Loading state. */
  loading: boolean;
  /** Error message if fetch failed. */
  error: string | null;
  /** Mark a single notification as read. */
  markRead: (id: string) => Promise<void>;
  /** Mark all notifications as read. */
  markAllRead: () => Promise<void>;
  /** Process a WS notification push event. */
  handleWSNotification: (msg: WSMessage<Notification>) => void;
  /** Refresh notifications from the server. */
  refresh: () => Promise<void>;
}

/** Hook to manage in-app notifications with WS push support. */
export function useNotifications({
  token,
  enabled = true,
}: UseNotificationsOptions): UseNotificationsReturn {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const unreadCount = notifications.filter((n) => !n.is_read).length;

  const fetchNotifications = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    setError(null);
    try {
      const url = buildUrl("/notifications", { limit: "50" });
      const response = await fetch(url, {
        method: "GET",
        headers: buildHeaders(token),
        cache: "no-store",
      });
      const result = await parseResponse<PaginatedResponse<Notification>>(response);
      setNotifications(result.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch notifications");
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    if (enabled && token) {
      void fetchNotifications();
    }
  }, [enabled, token, fetchNotifications]);

  const markRead = useCallback(
    async (id: string) => {
      if (!token) return;
      try {
        await clientMutate("PATCH", `/notifications/${id}/read`, { token });
        setNotifications((prev) => prev.map((n) => (n.id === id ? { ...n, is_read: true } : n)));
      } catch {
        // Silently fail — UI already updated optimistically.
      }
    },
    [token],
  );

  const markAllRead = useCallback(async () => {
    if (!token) return;
    try {
      await clientMutate("POST", "/notifications/read-all", { token });
      setNotifications((prev) => prev.map((n) => ({ ...n, is_read: true })));
    } catch {
      // Silently fail.
    }
  }, [token]);

  const handleWSNotification = useCallback((msg: WSMessage<Notification>) => {
    const notification = msg.payload;
    setNotifications((prev) => {
      // Avoid duplicates.
      if (prev.some((n) => n.id === notification.id)) return prev;
      return [notification, ...prev];
    });
  }, []);

  const refresh = useCallback(async () => {
    await fetchNotifications();
  }, [fetchNotifications]);

  return {
    notifications,
    unreadCount,
    loading,
    error,
    markRead,
    markAllRead,
    handleWSNotification,
    refresh,
  };
}
