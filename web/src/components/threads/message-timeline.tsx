"use client";

import { Mail, MessageSquare, Phone, Settings, StickyNote } from "lucide-react";
import type { Message, MessageType } from "@/lib/api-types";
import { cn } from "@/lib/utils";

export interface MessageTimelineProps {
  /** Messages to display, ordered by created_at. */
  messages: Message[];
  /** Called when the user clicks "Edit" on a message they authored. */
  onEdit?: (messageId: string) => void;
  /** Current user ID to show edit button on own messages. */
  currentUserId?: string;
}

const TYPE_ICONS: Record<MessageType, typeof MessageSquare> = {
  comment: MessageSquare,
  note: StickyNote,
  email: Mail,
  call_log: Phone,
  system: Settings,
};

const TYPE_LABELS: Record<MessageType, string> = {
  comment: "Comment",
  note: "Note",
  email: "Email",
  call_log: "Call Log",
  system: "System",
};

/** Format an ISO timestamp to a readable date string. */
function formatTimestamp(iso: string): string {
  try {
    const date = new Date(iso);
    return date.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return iso;
  }
}

/** Message timeline showing an ordered list of messages with type badges. */
export function MessageTimeline({
  messages,
  onEdit,
  currentUserId,
}: MessageTimelineProps): React.ReactNode {
  if (messages.length === 0) {
    return (
      <p className="py-6 text-center text-sm text-muted-foreground" data-testid="no-messages">
        No messages yet.
      </p>
    );
  }

  return (
    <div data-testid="message-timeline" className="flex flex-col gap-4">
      {messages.map((msg) => {
        const Icon = TYPE_ICONS[msg.type];
        const label = TYPE_LABELS[msg.type];
        const isAuthor = currentUserId !== undefined && msg.author_id === currentUserId;

        return (
          <div
            key={msg.id}
            data-testid={`message-${msg.id}`}
            className={cn(
              "rounded-lg border border-border p-4",
              msg.type === "system" && "bg-muted/50",
            )}
          >
            {/* Header */}
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Icon className="h-3.5 w-3.5" data-testid={`message-icon-${msg.id}`} />
              <span
                className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium"
                data-testid={`message-type-${msg.id}`}
              >
                {label}
              </span>
              <span data-testid={`message-author-${msg.id}`}>{msg.author_id}</span>
              <span>·</span>
              <time dateTime={msg.created_at} data-testid={`message-time-${msg.id}`}>
                {formatTimestamp(msg.created_at)}
              </time>
              {isAuthor && onEdit && (
                <button
                  onClick={() => onEdit(msg.id)}
                  data-testid={`message-edit-${msg.id}`}
                  className="ml-auto text-xs text-primary hover:underline"
                >
                  Edit
                </button>
              )}
            </div>
            {/* Body */}
            <div className="mt-2 text-sm text-foreground" data-testid={`message-body-${msg.id}`}>
              {msg.body}
            </div>
          </div>
        );
      })}
    </div>
  );
}
