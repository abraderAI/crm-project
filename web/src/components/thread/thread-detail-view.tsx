"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@clerk/nextjs";
import { AlertTriangle, History } from "lucide-react";

import type { Thread, Message, Revision as ApiRevision, Upload } from "@/lib/api-types";
import {
  createMessage,
  fetchThreadRevisions,
  fetchThreadUploads,
  uploadFile,
  toggleVote,
  createFlag,
} from "@/lib/entity-api";
import { ThreadDetail } from "./thread-detail";
import { VoteButton } from "@/components/community/vote-button";
import { FlagForm } from "@/components/community/flag-form";
import { MessageEditor } from "@/components/editor/message-editor";
import { RevisionHistory, type Revision } from "@/components/editor/revision-history";
import { FileUpload } from "@/components/upload/file-upload";
import { FileList, type FileItem } from "@/components/upload/file-list";

export interface ThreadDetailViewProps {
  /** Thread data. */
  thread: Thread;
  /** Messages for this thread. */
  messages: Message[];
  /** Parent org slug. */
  orgSlug: string;
  /** Parent space slug. */
  spaceSlug: string;
  /** Parent board slug. */
  boardSlug: string;
  /** Thread slug. */
  threadSlug: string;
  /** Whether the current user has voted on this thread. */
  hasVoted?: boolean;
}

/** Map API revision to component revision. */
function toRevision(r: ApiRevision): Revision {
  return {
    id: r.id,
    version: r.version,
    editorId: r.editor_id,
    previousContent: r.previous_content,
    createdAt: r.created_at,
  };
}

/** Map API upload to FileItem. */
function toFileItem(u: Upload): FileItem {
  const apiBase = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  return {
    id: u.id,
    filename: u.filename,
    contentType: u.content_type,
    size: u.size,
    downloadUrl: `${apiBase}/v1/uploads/${u.id}`,
  };
}

/** Client wrapper wiring ThreadDetail to MessageEditor, RevisionHistory, FileUpload. */
export function ThreadDetailView({
  thread,
  messages,
  orgSlug,
  spaceSlug,
  boardSlug,
  threadSlug,
  hasVoted = false,
}: ThreadDetailViewProps): React.ReactNode {
  const router = useRouter();
  const { getToken, userId } = useAuth();
  const [sending, setSending] = useState(false);
  const [voting, setVoting] = useState(false);
  const editorRef = useRef<HTMLDivElement>(null);

  // Flag form state
  const [showFlagForm, setShowFlagForm] = useState(false);
  const [flagging, setFlagging] = useState(false);

  // Revision history state
  const [showRevisions, setShowRevisions] = useState(false);
  const [revisions, setRevisions] = useState<Revision[]>([]);
  const [revisionsLoaded, setRevisionsLoaded] = useState(false);

  // File upload state
  const [files, setFiles] = useState<FileItem[]>([]);
  const [filesLoaded, setFilesLoaded] = useState(false);

  const loadRevisions = useCallback(async (): Promise<void> => {
    if (revisionsLoaded) return;
    const token = await getToken();
    if (!token) return;
    const { data } = await fetchThreadRevisions(token, orgSlug, spaceSlug, boardSlug, threadSlug);
    setRevisions(data.map(toRevision));
    setRevisionsLoaded(true);
  }, [getToken, orgSlug, spaceSlug, boardSlug, threadSlug, revisionsLoaded]);

  const loadFiles = useCallback(async (): Promise<void> => {
    if (filesLoaded) return;
    const token = await getToken();
    if (!token) return;
    const { data } = await fetchThreadUploads(token, orgSlug, spaceSlug, boardSlug, threadSlug);
    setFiles(data.map(toFileItem));
    setFilesLoaded(true);
  }, [getToken, orgSlug, spaceSlug, boardSlug, threadSlug, filesLoaded]);

  useEffect(() => {
    void loadFiles();
  }, [loadFiles]);

  const handleToggleRevisions = async (): Promise<void> => {
    if (!showRevisions) {
      await loadRevisions();
    }
    setShowRevisions(!showRevisions);
  };

  const handleSendMessage = async (content: string): Promise<void> => {
    if (!content.trim()) return;
    setSending(true);
    try {
      const token = await getToken();
      if (!token) throw new Error("Unauthenticated");
      await createMessage(token, orgSlug, spaceSlug, boardSlug, threadSlug, {
        body: content,
      });
      router.refresh();
    } finally {
      setSending(false);
    }
  };

  const handleFileUpload = async (selectedFiles: File[]): Promise<void> => {
    const token = await getToken();
    if (!token) return;
    for (const file of selectedFiles) {
      const upload = await uploadFile(token, orgSlug, spaceSlug, boardSlug, threadSlug, file);
      setFiles((prev) => [...prev, toFileItem(upload)]);
    }
  };

  const handleToggleVote = async (): Promise<void> => {
    setVoting(true);
    try {
      const token = await getToken();
      if (!token) throw new Error("Unauthenticated");
      await toggleVote(token, orgSlug, spaceSlug, boardSlug, threadSlug);
      router.refresh();
    } finally {
      setVoting(false);
    }
  };

  const handleSubmitFlag = async (reason: string): Promise<void> => {
    setFlagging(true);
    try {
      const token = await getToken();
      if (!token) throw new Error("Unauthenticated");
      await createFlag(token, thread.id, reason);
      setShowFlagForm(false);
    } finally {
      setFlagging(false);
    }
  };

  const handleScrollToEditor = (): void => {
    editorRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  const editor = !thread.is_locked ? (
    <div ref={editorRef}>
      <MessageEditor
        onSubmit={handleSendMessage}
        placeholder="Write a reply..."
        disabled={sending}
        submitLabel={sending ? "Sending..." : "Reply"}
      />
      <div className="mt-3">
        <FileUpload onUpload={handleFileUpload} multiple />
      </div>
    </div>
  ) : undefined;

  return (
    <div className="space-y-4">
      <VoteButton
        voteScore={thread.vote_score}
        hasVoted={hasVoted}
        onToggle={handleToggleVote}
        disabled={voting}
      />
      <ThreadDetail
        thread={thread}
        messages={messages}
        currentUserId={userId ?? undefined}
        onNewMessage={!thread.is_locked ? handleScrollToEditor : undefined}
        editorSlot={editor}
      />

      {/* Attachments */}
      {files.length > 0 && (
        <div data-testid="thread-attachments">
          <h3 className="mb-2 text-sm font-semibold text-foreground">
            Attachments ({files.length})
          </h3>
          <FileList files={files} />
        </div>
      )}

      {/* Flag content toggle */}
      <div>
        <button
          onClick={() => setShowFlagForm(!showFlagForm)}
          data-testid="flag-toggle"
          className="inline-flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <AlertTriangle className="h-4 w-4" />
          {showFlagForm ? "Cancel Report" : "Report Content"}
        </button>
        {showFlagForm && (
          <div className="mt-3">
            <FlagForm
              onSubmit={handleSubmitFlag}
              onCancel={() => setShowFlagForm(false)}
              loading={flagging}
            />
          </div>
        )}
      </div>

      {/* Revision History toggle */}
      <div>
        <button
          onClick={handleToggleRevisions}
          data-testid="revision-toggle"
          className="inline-flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <History className="h-4 w-4" />
          {showRevisions ? "Hide Revision History" : "Show Revision History"}
        </button>
        {showRevisions && (
          <div className="mt-3">
            <RevisionHistory revisions={revisions} />
          </div>
        )}
      </div>
    </div>
  );
}
