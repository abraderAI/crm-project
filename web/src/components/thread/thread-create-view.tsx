"use client";

import { useState, useRef } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@clerk/nextjs";

import { createThread } from "@/lib/entity-api";
import { MessageEditor } from "@/components/editor/message-editor";

export interface ThreadCreateViewProps {
  /** Parent org slug. */
  orgSlug: string;
  /** Parent space slug. */
  spaceSlug: string;
  /** Parent board slug. */
  boardSlug: string;
  /** Path to navigate to on cancel. */
  cancelHref: string;
}

/** Client component for creating a new thread with title and optional body. */
export function ThreadCreateView({
  orgSlug,
  spaceSlug,
  boardSlug,
  cancelHref,
}: ThreadCreateViewProps): React.ReactNode {
  const router = useRouter();
  const { getToken } = useAuth();
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [loading, setLoading] = useState(false);
  const bodyRef = useRef(body);
  bodyRef.current = body;

  const handleSubmit = async (): Promise<void> => {
    if (!title.trim()) return;
    setLoading(true);
    try {
      const token = await getToken();
      if (!token) throw new Error("Unauthenticated");
      const thread = await createThread(token, orgSlug, spaceSlug, boardSlug, {
        title: title.trim(),
        body: bodyRef.current || undefined,
        metadata: "{}",
      });
      router.push(
        `/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}/threads/${thread.slug}`,
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="mx-auto max-w-2xl space-y-4 p-6">
      <h1 className="text-lg font-bold text-foreground">New Thread</h1>

      {/* Title */}
      <div>
        <label htmlFor="thread-title" className="mb-1 block text-sm font-medium text-foreground">
          Title
        </label>
        <input
          id="thread-title"
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Thread title"
          disabled={loading}
          className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
        />
      </div>

      {/* Body (optional, rich editor) */}
      <div>
        <label className="mb-1 block text-sm font-medium text-foreground">Body (optional)</label>
        <MessageEditor
          onSubmit={() => handleSubmit()}
          onChange={(content) => setBody(content)}
          placeholder="Add details..."
          disabled={loading}
          showSubmit={false}
        />
      </div>

      {/* Actions */}
      <div className="flex justify-end gap-2">
        <button
          onClick={() => router.push(cancelHref)}
          disabled={loading}
          className="rounded-md border border-border px-4 py-2 text-sm text-foreground hover:bg-accent disabled:opacity-50"
        >
          Cancel
        </button>
        <button
          onClick={handleSubmit}
          disabled={loading || !title.trim()}
          className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          {loading ? "Creating..." : "Create Thread"}
        </button>
      </div>
    </div>
  );
}
