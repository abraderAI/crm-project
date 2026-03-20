"use client";

import { Mail, MessageSquare, Phone, FileText, Bot, User, Send, PenLine, Lock, Zap } from "lucide-react";
import type { AnyMessageType, Message } from "@/lib/api-types";
import { formatDate } from "./thread-list";

const TYPE_CONFIG: Partial<Record<AnyMessageType, { icon: typeof MessageSquare; label: string }>> = {
  // Generic types
  comment: { icon: MessageSquare, label: "Comment" },
  note: { icon: FileText, label: "Note" },
  email: { icon: Mail, label: "Email" },
  call_log: { icon: Phone, label: "Call" },
  system: { icon: Bot, label: "System" },
  // Support-specific types
  customer: { icon: User, label: "Customer" },
  agent_reply: { icon: Send, label: "Agent Reply" },
  draft: { icon: PenLine, label: "Draft" },
  context: { icon: Lock, label: "Internal" },
  system_event: { icon: Zap, label: "System Event" },
};

const DEFAULT_CONFIG = { icon: MessageSquare, label: "Message" };

export interface MessageTimelineProps {
  messages: Message[];
  /** Called when the edit button on a message is clicked. */
  onEdit?: (messageId: string) => void;
  /** Currently logged-in user ID for showing edit buttons. */
  currentUserId?: string;
}

/** Chronological timeline of messages with type badges and optional edit. */
export function MessageTimeline({
  messages,
  onEdit,
  currentUserId,
}: MessageTimelineProps): React.ReactNode {
  if (messages.length === 0) {
    return (
      <div
        className="py-6 text-center text-sm text-muted-foreground"
        data-testid="message-timeline-empty"
      >
        No messages yet.
      </div>
    );
  }

  return (
    <div className="space-y-4" data-testid="message-timeline">
      {messages.map((msg) => {
        const config = TYPE_CONFIG[msg.type] ?? DEFAULT_CONFIG;
        const Icon = config.icon;
        const isOwner = currentUserId === msg.author_id;

        return (
          <div
            key={msg.id}
            className="rounded-lg border border-border bg-background p-3"
            data-testid={`message-item-${msg.id}`}
          >
            <div className="mb-2 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Icon className="h-4 w-4 text-muted-foreground" />
                <span
                  className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground"
                  data-testid={`message-type-${msg.id}`}
                >
                  {config.label}
                </span>
                <span className="text-xs text-muted-foreground">{formatDate(msg.created_at)}</span>
              </div>
              {isOwner && onEdit && (
                <button
                  onClick={() => onEdit(msg.id)}
                  data-testid={`message-edit-${msg.id}`}
                  className="rounded-md px-2 py-0.5 text-xs text-primary hover:bg-accent"
                >
                  Edit
                </button>
              )}
            </div>
            <div
              className="prose prose-sm max-w-none text-foreground"
              data-testid={`message-body-${msg.id}`}
            >
              {msg.body}
            </div>
          </div>
        );
      })}
    </div>
  );
}
