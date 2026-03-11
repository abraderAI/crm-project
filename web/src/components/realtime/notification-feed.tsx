"use client";

import { CheckCheck, MessageSquare, AtSign, GitBranch, UserPlus } from "lucide-react";
import type { Notification } from "@/lib/api-types";
import { cn } from "@/lib/utils";

/** Icon mapping for notification types. */
const TYPE_ICONS: Record<string, typeof MessageSquare> = {
  message: MessageSquare,
  mention: AtSign,
  stage_change: GitBranch,
  assignment: UserPlus,
};

export interface NotificationFeedProps {
  /** Notifications to display. */
  notifications: Notification[];
  /** Loading state. */
  loading?: boolean;
  /** Called to mark a notification as read. */
  onMarkRead?: (id: string) => void;
  /** Called to mark all as read. */
  onMarkAllRead?: () => void;
}

/** Format a relative time string. */
export function formatRelativeTime(dateStr: string): string {
  const now = Date.now();
  const date = new Date(dateStr).getTime();
  const diffMs = now - date;
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 60) return "just now";
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDay = Math.floor(diffHr / 24);
  if (diffDay < 7) return `${diffDay}d ago`;
  return new Date(dateStr).toLocaleDateString();
}

/** Dropdown feed of recent notifications. */
export function NotificationFeed({
  notifications,
  loading = false,
  onMarkRead,
  onMarkAllRead,
}: NotificationFeedProps): React.ReactNode {
  const hasUnread = notifications.some((n) => !n.is_read);

  return (
    <div
      className="w-80 rounded-lg border border-border bg-background shadow-lg"
      data-testid="notification-feed"
    >
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <h3 className="text-sm font-semibold text-foreground">Notifications</h3>
        {hasUnread && onMarkAllRead && (
          <button
            onClick={onMarkAllRead}
            className="flex items-center gap-1 text-xs text-primary hover:underline"
            data-testid="mark-all-read-btn"
          >
            <CheckCheck className="h-3 w-3" />
            Mark all read
          </button>
        )}
      </div>

      {/* Body */}
      <div className="max-h-96 overflow-y-auto" data-testid="notification-feed-list">
        {loading && notifications.length === 0 && (
          <div
            className="px-4 py-8 text-center text-sm text-muted-foreground"
            data-testid="notification-feed-loading"
          >
            Loading notifications...
          </div>
        )}
        {!loading && notifications.length === 0 && (
          <div
            className="px-4 py-8 text-center text-sm text-muted-foreground"
            data-testid="notification-feed-empty"
          >
            No notifications yet.
          </div>
        )}
        {notifications.map((notif) => {
          const Icon = TYPE_ICONS[notif.type] ?? MessageSquare;
          return (
            <button
              key={notif.id}
              onClick={() => onMarkRead?.(notif.id)}
              className={cn(
                "flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-muted/50",
                !notif.is_read && "bg-primary/5",
              )}
              data-testid={`notification-item-${notif.id}`}
            >
              <Icon className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span
                    className={cn(
                      "truncate text-sm",
                      notif.is_read ? "text-muted-foreground" : "font-medium text-foreground",
                    )}
                    data-testid={`notification-title-${notif.id}`}
                  >
                    {notif.title}
                  </span>
                  {!notif.is_read && (
                    <span
                      className="h-2 w-2 shrink-0 rounded-full bg-primary"
                      data-testid={`notification-unread-dot-${notif.id}`}
                    />
                  )}
                </div>
                {notif.body && (
                  <p
                    className="mt-0.5 truncate text-xs text-muted-foreground"
                    data-testid={`notification-body-${notif.id}`}
                  >
                    {notif.body}
                  </p>
                )}
                <span className="mt-1 text-xs text-muted-foreground/70">
                  {formatRelativeTime(notif.created_at)}
                </span>
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}
