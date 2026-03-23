"use client";

import { useState, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useAuth } from "@clerk/nextjs";
import { AlertTriangle, ArrowLeft, MessageSquare } from "lucide-react";

import { createForumThread } from "@/lib/global-api";
import { MessageEditor } from "@/components/editor/message-editor";

/**
 * New forum thread creation page at /forum/new.
 * Requires authentication (middleware-protected).
 */
export default function NewForumThreadPage(): ReactNode {
  const router = useRouter();
  const { getToken } = useAuth();

  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const handleCreate = async (): Promise<void> => {
    if (!title.trim()) return;
    setError("");
    setSubmitting(true);
    try {
      const token = await getToken();
      if (!token) return;
      const thread = await createForumThread(token, {
        title: title.trim(),
        body: body.trim() || undefined,
      });
      router.push(`/forum/${thread.slug}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create thread");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div data-testid="new-forum-thread-page" className="mx-auto max-w-2xl px-4 py-8">
      {/* Header */}
      <div className="mb-6 flex flex-col gap-2">
        <Link
          href="/forum"
          data-testid="back-to-forum-link"
          className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-3 w-3" />
          Back to forum
        </Link>
        <div className="flex items-center gap-3">
          <MessageSquare className="h-6 w-6 text-primary" />
          <h1 className="text-xl font-semibold text-foreground">New Thread</h1>
        </div>
      </div>

      <div className="flex flex-col gap-5 rounded-xl border border-border bg-background p-6 shadow-sm">
        {/* Title */}
        <div>
          <label htmlFor="forum-thread-title" className="text-xs font-medium text-foreground">
            Title <span className="text-red-500">*</span>
          </label>
          <input
            id="forum-thread-title"
            data-testid="forum-thread-title"
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="What would you like to discuss?"
            className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        {/* Body */}
        <div>
          <label className="text-xs font-medium text-foreground">Body</label>
          <div className="mt-1">
            <MessageEditor
              initialContent={body}
              onSubmit={() => void handleCreate()}
              onChange={setBody}
              placeholder="Share your thoughts, ask a question, or start a discussion..."
              disabled={submitting}
              showSubmit={false}
            />
          </div>
        </div>

        {/* Error */}
        {error && (
          <div
            data-testid="forum-thread-error"
            className="flex items-center gap-2 rounded-md bg-red-50 px-3 py-2 text-sm text-red-700"
          >
            <AlertTriangle className="h-4 w-4 shrink-0" />
            {error}
          </div>
        )}

        {/* Actions */}
        <div className="flex justify-end gap-3">
          <button
            onClick={() => router.push("/forum")}
            className="rounded-md border border-border px-4 py-2 text-sm text-foreground hover:bg-accent"
          >
            Cancel
          </button>
          <button
            data-testid="forum-thread-submit"
            onClick={() => void handleCreate()}
            disabled={submitting || !title.trim()}
            className="inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {submitting ? "Posting…" : "Post Thread"}
          </button>
        </div>
      </div>
    </div>
  );
}
