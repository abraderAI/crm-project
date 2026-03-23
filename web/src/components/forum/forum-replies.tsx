"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { useAuth } from "@clerk/nextjs";
import { MessageSquare, Send } from "lucide-react";

import type { MessageWithAuthor } from "@/lib/api-types";
import { createForumReply, fetchForumMessages } from "@/lib/global-api";
import { AuthorAvatar } from "./author-avatar";
import { relativeTime } from "./relative-time";

interface ForumRepliesProps {
  threadSlug: string;
  isLocked: boolean;
}

/** Client component that loads and displays replies, with an inline reply form. */
export function ForumReplies({ threadSlug, isLocked }: ForumRepliesProps): ReactNode {
  const { getToken, isSignedIn } = useAuth();
  const [messages, setMessages] = useState<MessageWithAuthor[]>([]);
  const [loading, setLoading] = useState(true);
  const [replyBody, setReplyBody] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const mountedRef = useRef(true);

  const loadMessages = useCallback(async () => {
    setLoading(true);
    try {
      const result = await fetchForumMessages(threadSlug, { limit: 100 });
      if (!mountedRef.current) return;
      setMessages(result.data);
    } catch {
      // Non-critical — show empty.
    } finally {
      if (mountedRef.current) setLoading(false);
    }
  }, [threadSlug]);

  useEffect(() => {
    mountedRef.current = true;
    void loadMessages();
    return () => {
      mountedRef.current = false;
    };
  }, [loadMessages]);

  const handleSubmitReply = async (): Promise<void> => {
    if (!replyBody.trim() || submitting) return;
    setError("");
    setSubmitting(true);
    try {
      const token = await getToken();
      if (!token) return;
      await createForumReply(token, threadSlug, replyBody.trim());
      setReplyBody("");
      void loadMessages();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to post reply");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div data-testid="forum-replies" className="space-y-4">
      <h2 className="flex items-center gap-2 text-sm font-semibold text-foreground">
        <MessageSquare className="h-4 w-4 text-primary" />
        Replies
        {messages.length > 0 && (
          <span className="text-xs font-normal text-muted-foreground">({messages.length})</span>
        )}
      </h2>

      {/* Reply list */}
      {loading ? (
        <div className="animate-pulse space-y-3">
          {Array.from({ length: 2 }).map((_, i) => (
            <div key={i} className="h-16 rounded-lg bg-muted" />
          ))}
        </div>
      ) : messages.length === 0 ? (
        <p data-testid="no-replies" className="text-sm text-muted-foreground">
          No replies yet. Be the first to respond!
        </p>
      ) : (
        <div className="space-y-3">
          {messages.map((msg) => (
            <div
              key={msg.id}
              data-testid={`reply-${msg.id}`}
              className="rounded-xl border border-border bg-background p-4 shadow-sm"
            >
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <AuthorAvatar authorId={msg.author_id} authorName={msg.author_name} size="sm" />
                <span className="font-medium text-foreground">
                  {msg.author_name || msg.author_id.slice(0, 12)}
                </span>
                {msg.author_org && (
                  <span className="rounded-full bg-primary/10 px-2 py-0.5 text-[10px] font-medium text-primary">
                    {msg.author_org}
                  </span>
                )}
                <span>·</span>
                <span>{relativeTime(msg.created_at)}</span>
              </div>
              <div className="mt-2 text-sm text-foreground whitespace-pre-wrap">{msg.body}</div>
            </div>
          ))}
        </div>
      )}

      {/* Reply form */}
      {isLocked ? (
        <p className="text-xs text-muted-foreground italic">This thread is locked.</p>
      ) : isSignedIn ? (
        <div data-testid="reply-form" className="space-y-2">
          <textarea
            data-testid="reply-textarea"
            value={replyBody}
            onChange={(e) => setReplyBody(e.target.value)}
            placeholder="Write a reply..."
            disabled={submitting}
            rows={3}
            className="w-full rounded-xl border border-border bg-background px-4 py-3 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary resize-none"
          />
          {error && <p className="text-xs text-destructive">{error}</p>}
          <div className="flex justify-end">
            <button
              data-testid="reply-submit"
              onClick={() => void handleSubmitReply()}
              disabled={submitting || !replyBody.trim()}
              className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors"
            >
              <Send className="h-3.5 w-3.5" />
              {submitting ? "Posting…" : "Reply"}
            </button>
          </div>
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">
          <a href="/sign-in?redirect_url=/forum" className="text-primary hover:underline">
            Sign in
          </a>{" "}
          to reply.
        </p>
      )}
    </div>
  );
}
