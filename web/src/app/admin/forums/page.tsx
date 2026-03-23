"use client";

import { useCallback, useEffect, useState, type ReactNode } from "react";
import { useAuth } from "@clerk/nextjs";
import { Eye, EyeOff, Lock, MessageSquare, Pin, PinOff, Unlock } from "lucide-react";

import type { Thread } from "@/lib/api-types";
import {
  fetchAdminForumThreads,
  toggleForumThreadHidden,
  toggleForumThreadLocked,
  toggleForumThreadPin,
} from "@/lib/admin-forum-api";
import { relativeTime } from "@/components/forum/relative-time";

/** Admin forums management page at /admin/forums. */
export default function AdminForumsPage(): ReactNode {
  const { getToken } = useAuth();
  const [threads, setThreads] = useState<Thread[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const token = await getToken();
      if (!token) return;
      const result = await fetchAdminForumThreads(token, { limit: 50 });
      setThreads(result.data);
    } catch {
      // Silently fail — admin will see empty state.
    } finally {
      setLoading(false);
    }
  }, [getToken]);

  useEffect(() => {
    void load();
  }, [load]);

  const handleAction = async (action: "pin" | "hide" | "lock", thread: Thread): Promise<void> => {
    const token = await getToken();
    if (!token) return;
    try {
      switch (action) {
        case "pin":
          await toggleForumThreadPin(token, thread.slug, !thread.is_pinned);
          break;
        case "hide":
          await toggleForumThreadHidden(token, thread.slug, !thread.is_hidden);
          break;
        case "lock":
          await toggleForumThreadLocked(token, thread.slug, !thread.is_locked);
          break;
      }
      void load();
    } catch {
      // Best effort — reload will show current state.
    }
  };

  if (loading) {
    return (
      <div data-testid="admin-forums-loading" className="animate-pulse space-y-3">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="h-12 rounded-lg bg-muted" />
        ))}
      </div>
    );
  }

  return (
    <div data-testid="admin-forums-page" className="space-y-6">
      <div className="flex items-center gap-3">
        <MessageSquare className="h-5 w-5 text-primary" />
        <h2 className="text-lg font-semibold text-foreground">Forum Management</h2>
        <span className="text-sm text-muted-foreground">
          {threads.length} {threads.length === 1 ? "thread" : "threads"}
        </span>
      </div>

      {threads.length === 0 ? (
        <p data-testid="admin-forums-empty" className="text-sm text-muted-foreground">
          No forum threads found.
        </p>
      ) : (
        <div className="divide-y divide-border rounded-lg border border-border">
          {threads.map((thread) => (
            <div
              key={thread.id}
              data-testid={`admin-forum-thread-${thread.id}`}
              className="flex items-center gap-4 px-4 py-3"
            >
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  {thread.is_pinned && <Pin className="h-3 w-3 text-amber-500" />}
                  {thread.is_hidden && <EyeOff className="h-3 w-3 text-red-500" />}
                  {thread.is_locked && <Lock className="h-3 w-3 text-muted-foreground" />}
                  <span className="text-sm font-medium text-foreground truncate">
                    {thread.title}
                  </span>
                </div>
                <p className="text-xs text-muted-foreground">
                  {relativeTime(thread.created_at)} · ▲ {thread.vote_score}
                </p>
              </div>

              <div className="flex shrink-0 items-center gap-1">
                <button
                  data-testid={`action-pin-${thread.id}`}
                  onClick={() => void handleAction("pin", thread)}
                  title={thread.is_pinned ? "Unpin" : "Pin"}
                  className="rounded p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
                >
                  {thread.is_pinned ? <PinOff className="h-4 w-4" /> : <Pin className="h-4 w-4" />}
                </button>
                <button
                  data-testid={`action-hide-${thread.id}`}
                  onClick={() => void handleAction("hide", thread)}
                  title={thread.is_hidden ? "Unhide" : "Hide"}
                  className="rounded p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
                >
                  {thread.is_hidden ? <Eye className="h-4 w-4" /> : <EyeOff className="h-4 w-4" />}
                </button>
                <button
                  data-testid={`action-lock-${thread.id}`}
                  onClick={() => void handleAction("lock", thread)}
                  title={thread.is_locked ? "Unlock" : "Lock"}
                  className="rounded p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
                >
                  {thread.is_locked ? <Unlock className="h-4 w-4" /> : <Lock className="h-4 w-4" />}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
