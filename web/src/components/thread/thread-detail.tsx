"use client";

import type { Thread, Message } from "@/lib/api-types";
import { MetadataSidebar } from "./metadata-sidebar";
import { MessageTimeline } from "./message-timeline";

export interface ThreadDetailProps {
  thread: Thread;
  messages: Message[];
  currentUserId?: string;
  onEditMessage?: (messageId: string) => void;
  onNewMessage?: () => void;
  /** Render slot for the message editor. */
  editorSlot?: React.ReactNode;
}

/** Full thread detail view: title, body, metadata sidebar, message timeline. */
export function ThreadDetail({
  thread,
  messages,
  currentUserId,
  onEditMessage,
  onNewMessage,
  editorSlot,
}: ThreadDetailProps): React.ReactNode {
  return (
    <div className="flex gap-6" data-testid="thread-detail">
      {/* Main content */}
      <div className="min-w-0 flex-1">
        <h1 className="text-xl font-bold text-foreground" data-testid="thread-title">
          {thread.title}
        </h1>
        {thread.body && (
          <div className="prose prose-sm mt-3 max-w-none text-foreground" data-testid="thread-body">
            {thread.body}
          </div>
        )}

        {/* Messages */}
        <div className="mt-6">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-foreground">Messages ({messages.length})</h2>
            {onNewMessage && !thread.is_locked && (
              <button
                onClick={onNewMessage}
                data-testid="thread-new-message-btn"
                className="rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90"
              >
                New message
              </button>
            )}
          </div>
          <MessageTimeline
            messages={messages}
            currentUserId={currentUserId}
            onEdit={onEditMessage}
          />
        </div>

        {/* Editor slot */}
        {editorSlot && (
          <div className="mt-4" data-testid="thread-editor-slot">
            {editorSlot}
          </div>
        )}

        {/* Locked notice */}
        {thread.is_locked && (
          <div
            className="mt-4 rounded-md bg-muted p-3 text-center text-sm text-muted-foreground"
            data-testid="thread-locked-notice"
          >
            This thread is locked. New messages cannot be added.
          </div>
        )}
      </div>

      {/* Sidebar */}
      <MetadataSidebar
        status={thread.status}
        priority={thread.priority}
        stage={thread.stage}
        assignedTo={thread.assigned_to}
        voteScore={thread.vote_score}
        metadata={thread.metadata}
        isPinned={thread.is_pinned}
        isLocked={thread.is_locked}
      />
    </div>
  );
}
