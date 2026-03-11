"use client";

import { Lock, Pin } from "lucide-react";
import type { Thread, Message } from "@/lib/api-types";
import { MessageTimeline } from "./message-timeline";

export interface ThreadDetailProps {
  /** The thread to display. */
  thread: Thread;
  /** Messages in this thread. */
  messages: Message[];
  /** Current user ID for edit button visibility. */
  currentUserId?: string;
  /** Called when the user clicks "Edit" on a message. */
  onEditMessage?: (messageId: string) => void;
}

/** Parse metadata JSON safely for sidebar display. */
function parseMetadata(metadata: string): Record<string, unknown> {
  try {
    const parsed: unknown = JSON.parse(metadata);
    if (typeof parsed === "object" && parsed !== null && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
    return {};
  } catch {
    return {};
  }
}

/** Thread detail view with title, body, metadata sidebar, and message timeline. */
export function ThreadDetail({
  thread,
  messages,
  currentUserId,
  onEditMessage,
}: ThreadDetailProps): React.ReactNode {
  const meta = parseMetadata(thread.metadata);

  return (
    <div data-testid="thread-detail" className="flex gap-6">
      {/* Main content */}
      <div className="flex-1 min-w-0">
        {/* Title */}
        <div className="flex items-center gap-2">
          <h1 className="text-xl font-bold text-foreground">{thread.title}</h1>
          {thread.is_pinned && <Pin className="h-4 w-4 text-primary" data-testid="thread-pin" />}
          {thread.is_locked && (
            <Lock className="h-4 w-4 text-muted-foreground" data-testid="thread-lock" />
          )}
        </div>

        {/* Body */}
        {thread.body && (
          <div
            className="mt-4 rounded-lg border border-border p-4 text-sm text-foreground"
            data-testid="thread-body"
          >
            {thread.body}
          </div>
        )}

        {/* Messages */}
        <div className="mt-6">
          <h2 className="mb-3 text-sm font-semibold text-foreground">Messages</h2>
          <MessageTimeline
            messages={messages}
            currentUserId={currentUserId}
            onEdit={onEditMessage}
          />
        </div>
      </div>

      {/* Metadata sidebar */}
      <aside
        data-testid="thread-sidebar"
        className="w-64 shrink-0 rounded-lg border border-border p-4"
      >
        <h3 className="text-sm font-semibold text-foreground">Details</h3>
        <dl className="mt-3 flex flex-col gap-2 text-sm">
          {thread.status && (
            <div>
              <dt className="text-xs text-muted-foreground">Status</dt>
              <dd className="font-medium text-foreground" data-testid="sidebar-status">
                {thread.status}
              </dd>
            </div>
          )}
          {thread.priority && (
            <div>
              <dt className="text-xs text-muted-foreground">Priority</dt>
              <dd className="font-medium text-foreground" data-testid="sidebar-priority">
                {thread.priority}
              </dd>
            </div>
          )}
          {thread.stage && (
            <div>
              <dt className="text-xs text-muted-foreground">Stage</dt>
              <dd className="font-medium text-foreground" data-testid="sidebar-stage">
                {thread.stage}
              </dd>
            </div>
          )}
          {thread.assigned_to && (
            <div>
              <dt className="text-xs text-muted-foreground">Assigned To</dt>
              <dd className="font-medium text-foreground" data-testid="sidebar-assigned">
                {thread.assigned_to}
              </dd>
            </div>
          )}
          <div>
            <dt className="text-xs text-muted-foreground">Votes</dt>
            <dd className="font-medium text-foreground" data-testid="sidebar-votes">
              {thread.vote_score}
            </dd>
          </div>
          <div>
            <dt className="text-xs text-muted-foreground">Author</dt>
            <dd className="font-medium text-foreground" data-testid="sidebar-author">
              {thread.author_id}
            </dd>
          </div>
          {/* Custom metadata */}
          {Object.entries(meta).length > 0 && (
            <>
              <hr className="border-border" />
              <h4 className="text-xs font-semibold text-muted-foreground">Metadata</h4>
              {Object.entries(meta).map(([key, value]) => (
                <div key={key}>
                  <dt className="text-xs text-muted-foreground">{key}</dt>
                  <dd className="font-medium text-foreground" data-testid={`meta-${key}`}>
                    {String(value)}
                  </dd>
                </div>
              ))}
            </>
          )}
        </dl>
      </aside>
    </div>
  );
}
