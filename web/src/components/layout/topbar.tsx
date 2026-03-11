"use client";

import { Bell, Search } from "lucide-react";
import { useState } from "react";
import { ThemeToggle } from "@/components/theme-toggle";
import { cn } from "@/lib/utils";

interface TopbarProps {
  /** Number of unread notifications. */
  unreadCount?: number;
  /** Render slot for user menu (e.g. Clerk UserButton). */
  userMenu?: React.ReactNode;
  /** Called when search is submitted. */
  onSearch?: (query: string) => void;
}

/** Top navigation bar with search, notifications, theme toggle, and user menu. */
export function Topbar({ unreadCount = 0, userMenu, onSearch }: TopbarProps): React.ReactNode {
  const [searchValue, setSearchValue] = useState("");

  const handleSubmit = (e: React.FormEvent): void => {
    e.preventDefault();
    onSearch?.(searchValue);
  };

  return (
    <header
      data-testid="topbar"
      className="flex h-14 items-center justify-between border-b border-foreground/10 bg-background px-4"
    >
      {/* Search */}
      <form onSubmit={handleSubmit} className="flex w-full max-w-md items-center gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-foreground/40" />
          <input
            type="search"
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            placeholder="Search..."
            aria-label="Search"
            data-testid="search-input"
            className={cn(
              "h-9 w-full rounded-md border border-foreground/10 bg-foreground/5 pl-8 pr-3 text-sm",
              "placeholder:text-foreground/40 focus:border-foreground/20 focus:outline-none focus:ring-1 focus:ring-foreground/20",
            )}
          />
        </div>
      </form>

      {/* Right actions */}
      <div className="flex items-center gap-1">
        {/* Notification bell */}
        <button
          aria-label={
            unreadCount > 0 ? `${unreadCount} unread notifications` : "No unread notifications"
          }
          className="relative inline-flex items-center justify-center rounded-md p-2 text-foreground/70 transition-colors hover:bg-foreground/10 hover:text-foreground"
          data-testid="notification-bell"
        >
          <Bell className="h-5 w-5" />
          {unreadCount > 0 && (
            <span
              className="absolute right-1 top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white"
              data-testid="notification-badge"
            >
              {unreadCount > 99 ? "99+" : unreadCount}
            </span>
          )}
        </button>

        {/* Theme toggle */}
        <ThemeToggle />

        {/* User menu slot */}
        {userMenu && <div data-testid="user-menu">{userMenu}</div>}
      </div>
    </header>
  );
}
