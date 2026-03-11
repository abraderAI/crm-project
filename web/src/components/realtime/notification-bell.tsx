"use client";

import { Bell } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import type { Notification } from "@/lib/api-types";
import { NotificationFeed } from "./notification-feed";

export interface NotificationBellProps {
  /** Notifications to display. */
  notifications: Notification[];
  /** Number of unread notifications. */
  unreadCount: number;
  /** Loading state. */
  loading?: boolean;
  /** Called to mark a notification as read. */
  onMarkRead?: (id: string) => void;
  /** Called to mark all as read. */
  onMarkAllRead?: () => void;
}

/** Notification bell button with dropdown feed. */
export function NotificationBell({
  notifications,
  unreadCount,
  loading = false,
  onMarkRead,
  onMarkAllRead,
}: NotificationBellProps): React.ReactNode {
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const handleToggle = useCallback(() => {
    setIsOpen((prev) => !prev);
  }, []);

  // Close on outside click.
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent): void => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    }
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [isOpen]);

  // Close on Escape key.
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent): void => {
      if (e.key === "Escape") setIsOpen(false);
    };
    if (isOpen) {
      document.addEventListener("keydown", handleKeyDown);
    }
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOpen]);

  return (
    <div className="relative" ref={containerRef} data-testid="notification-bell-container">
      <button
        onClick={handleToggle}
        aria-label={
          unreadCount > 0 ? `${unreadCount} unread notifications` : "No unread notifications"
        }
        aria-expanded={isOpen}
        className="relative inline-flex items-center justify-center rounded-md p-2 text-foreground/70 transition-colors hover:bg-foreground/10 hover:text-foreground"
        data-testid="notification-bell-btn"
      >
        <Bell className="h-5 w-5" />
        {unreadCount > 0 && (
          <span
            className="absolute right-1 top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white"
            data-testid="notification-bell-badge"
          >
            {unreadCount > 99 ? "99+" : unreadCount}
          </span>
        )}
      </button>

      {isOpen && (
        <div className="absolute right-0 top-full z-50 mt-2" data-testid="notification-dropdown">
          <NotificationFeed
            notifications={notifications}
            loading={loading}
            onMarkRead={onMarkRead}
            onMarkAllRead={onMarkAllRead}
          />
        </div>
      )}
    </div>
  );
}
