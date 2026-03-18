import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";
import type { Thread, Message } from "@/lib/api-types";

// Mock Clerk auth.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken, userId: "user-1" }),
}));

// Mock next/navigation.
const mockRefresh = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ refresh: mockRefresh }),
}));

// Mock useWebSocket hook.
const mockSubscribe = vi.fn();
const mockUnsubscribe = vi.fn();
const mockWsSend = vi.fn();
vi.mock("@/hooks/use-websocket", () => ({
  useWebSocket: () => ({
    state: "connected" as const,
    subscribe: mockSubscribe,
    unsubscribe: mockUnsubscribe,
    send: mockWsSend,
    lastMessage: null,
  }),
}));

// Mock useTyping hook.
const mockHandleLocalTyping = vi.fn();
const mockHandleRemoteTyping = vi.fn();
vi.mock("@/hooks/use-typing", () => ({
  useTyping: () => ({
    typingUsers: [],
    handleLocalTyping: mockHandleLocalTyping,
    handleRemoteTyping: mockHandleRemoteTyping,
  }),
}));

// Mock entity-api.
const mockToggleVote = vi.fn();
const mockCreateFlag = vi.fn();
const mockCreateMessage = vi.fn();
const mockFetchThreadRevisions = vi.fn();
const mockFetchThreadUploads = vi.fn();
const mockUploadFile = vi.fn();
vi.mock("@/lib/entity-api", () => ({
  toggleVote: (...args: unknown[]) => mockToggleVote(...args),
  createFlag: (...args: unknown[]) => mockCreateFlag(...args),
  createMessage: (...args: unknown[]) => mockCreateMessage(...args),
  fetchThreadRevisions: (...args: unknown[]) => mockFetchThreadRevisions(...args),
  fetchThreadUploads: (...args: unknown[]) => mockFetchThreadUploads(...args),
  uploadFile: (...args: unknown[]) => mockUploadFile(...args),
}));

import { ThreadDetailView } from "./thread-detail-view";

const thread: Thread = {
  id: "t1",
  board_id: "b1",
  title: "Test Thread",
  body: "Thread body",
  slug: "test-thread",
  metadata: "{}",
  author_id: "user-1",
  is_pinned: false,
  is_locked: false,
  is_hidden: false,
  vote_score: 5,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const messages: Message[] = [
  {
    id: "m1",
    thread_id: "t1",
    body: "First message",
    author_id: "user-2",
    metadata: "{}",
    type: "comment",
    created_at: "2026-01-01T01:00:00Z",
    updated_at: "2026-01-01T01:00:00Z",
  },
];

const defaultProps = {
  thread,
  messages,
  orgSlug: "acme",
  spaceSlug: "community",
  boardSlug: "general",
  threadSlug: "test-thread",
};

describe("ThreadDetailView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
    mockFetchThreadUploads.mockResolvedValue({ data: [], page_info: { has_more: false } });
    mockFetchThreadRevisions.mockResolvedValue({ data: [], page_info: { has_more: false } });
    mockToggleVote.mockResolvedValue({ id: "v1", thread_id: "t1", user_id: "user-1", weight: 1 });
    mockCreateFlag.mockResolvedValue({ id: "f1", thread_id: "t1", reason: "Spam" });
  });

  it("renders VoteButton with correct initial state when hasVoted is true", () => {
    render(<ThreadDetailView {...defaultProps} hasVoted />);

    const voteButton = screen.getByTestId("vote-button");
    expect(voteButton).toBeInTheDocument();
    expect(screen.getByTestId("vote-score")).toHaveTextContent("5");
  });

  it("renders VoteButton with hasVoted false by default", () => {
    render(<ThreadDetailView {...defaultProps} />);

    const voteButton = screen.getByTestId("vote-button");
    expect(voteButton).toBeInTheDocument();
  });

  it("calls toggleVote via entity-api when VoteButton is clicked", async () => {
    const user = userEvent.setup();
    render(<ThreadDetailView {...defaultProps} hasVoted={false} />);

    await user.click(screen.getByTestId("vote-button"));

    await waitFor(() => {
      expect(mockToggleVote).toHaveBeenCalledWith(
        "test-token",
        "acme",
        "community",
        "general",
        "test-thread",
      );
    });
  });

  it("renders ThreadDetail with thread data", () => {
    render(<ThreadDetailView {...defaultProps} />);

    expect(screen.getByTestId("thread-detail")).toBeInTheDocument();
    expect(screen.getByTestId("thread-title")).toHaveTextContent("Test Thread");
  });

  it("renders revision toggle button", () => {
    render(<ThreadDetailView {...defaultProps} />);

    expect(screen.getByTestId("revision-toggle")).toBeInTheDocument();
  });

  it("shows flag toggle button", () => {
    render(<ThreadDetailView {...defaultProps} />);

    expect(screen.getByTestId("flag-toggle")).toBeInTheDocument();
  });

  it("shows FlagForm when flag toggle is clicked", async () => {
    const user = userEvent.setup();
    render(<ThreadDetailView {...defaultProps} />);

    await user.click(screen.getByTestId("flag-toggle"));

    expect(screen.getByTestId("flag-form")).toBeInTheDocument();
  });

  it("calls createFlag via entity-api when FlagForm is submitted", async () => {
    const user = userEvent.setup();
    render(<ThreadDetailView {...defaultProps} />);

    await user.click(screen.getByTestId("flag-toggle"));
    await user.click(screen.getByTestId("flag-reason-spam-or-misleading"));
    await user.click(screen.getByTestId("flag-submit-btn"));

    await waitFor(() => {
      expect(mockCreateFlag).toHaveBeenCalledWith("test-token", "t1", "Spam or misleading");
    });
  });

  it("renders RealtimeMessages instead of static MessageTimeline", () => {
    render(<ThreadDetailView {...defaultProps} />);
    expect(screen.getByTestId("realtime-messages")).toBeInTheDocument();
  });

  it("passes initial messages to RealtimeMessages", () => {
    render(<ThreadDetailView {...defaultProps} />);
    // RealtimeMessages renders MessageTimeline with initialMessages
    expect(screen.getByTestId("message-item-m1")).toBeInTheDocument();
  });

  it("subscribes to thread WS channel via RealtimeMessages", async () => {
    render(<ThreadDetailView {...defaultProps} />);
    await waitFor(() => {
      expect(mockSubscribe).toHaveBeenCalledWith("thread:t1");
    });
  });

  it("shows UploadProgress during file upload", async () => {
    let resolveUpload!: (value: unknown) => void;
    mockUploadFile.mockReturnValue(
      new Promise((resolve) => {
        resolveUpload = resolve;
      }),
    );

    const user = userEvent.setup();
    render(<ThreadDetailView {...defaultProps} />);

    const file = new File(["content"], "test.txt", { type: "text/plain" });
    const input = screen.getByTestId("file-input");
    await user.upload(input, file);

    // UploadProgress should appear while upload is pending
    await waitFor(() => {
      expect(screen.getByTestId("upload-progress")).toBeInTheDocument();
    });
    expect(screen.getByTestId("upload-filename-test.txt")).toHaveTextContent("test.txt");
    expect(screen.getByTestId("upload-percent-test.txt")).toHaveTextContent("0%");

    // Resolve the upload
    resolveUpload({
      id: "u1",
      org_id: "o1",
      entity_type: "thread",
      entity_id: "t1",
      filename: "test.txt",
      content_type: "text/plain",
      size: 7,
      storage_path: "/uploads/test.txt",
      uploader_id: "user-1",
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    });

    // Progress should update to 100%
    await waitFor(() => {
      expect(screen.getByTestId("upload-percent-test.txt")).toHaveTextContent("100%");
    });
  });

  it("shows upload error in UploadProgress when upload fails", async () => {
    mockUploadFile.mockRejectedValue(new Error("Network error"));

    const user = userEvent.setup();
    render(<ThreadDetailView {...defaultProps} />);

    const file = new File(["content"], "bad.txt", { type: "text/plain" });
    const input = screen.getByTestId("file-input");
    await user.upload(input, file);

    await waitFor(() => {
      expect(screen.getByTestId("upload-error-bad.txt")).toBeInTheDocument();
    });
    expect(screen.getByTestId("upload-error-bad.txt")).toHaveTextContent("Network error");
  });

  it("does not render editor when thread is locked", () => {
    render(<ThreadDetailView {...defaultProps} thread={{ ...thread, is_locked: true }} />);
    expect(screen.queryByTestId("message-editor")).not.toBeInTheDocument();
  });

  it("shows revision history when revision toggle is clicked", async () => {
    const user = userEvent.setup();
    render(<ThreadDetailView {...defaultProps} />);

    await user.click(screen.getByTestId("revision-toggle"));

    await waitFor(() => {
      expect(screen.getByTestId("no-revisions")).toBeInTheDocument();
    });
  });

  it("hides revision history when revision toggle is clicked twice", async () => {
    const user = userEvent.setup();
    render(<ThreadDetailView {...defaultProps} />);

    await user.click(screen.getByTestId("revision-toggle"));
    await waitFor(() => {
      expect(screen.getByTestId("no-revisions")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("revision-toggle"));
    expect(screen.queryByTestId("no-revisions")).not.toBeInTheDocument();
    expect(screen.queryByTestId("revision-history")).not.toBeInTheDocument();
  });

  it("displays file attachments when uploads are present", async () => {
    const upload = {
      id: "u1",
      org_id: "o1",
      entity_type: "thread",
      entity_id: "t1",
      filename: "document.pdf",
      content_type: "application/pdf",
      size: 1024,
      storage_path: "/uploads/document.pdf",
      uploader_id: "user-1",
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    };
    mockFetchThreadUploads.mockResolvedValue({ data: [upload], page_info: { has_more: false } });

    render(<ThreadDetailView {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getByTestId("thread-attachments")).toBeInTheDocument();
    });
  });

  it("does not send message when content is empty", async () => {
    const user = userEvent.setup();
    render(<ThreadDetailView {...defaultProps} />);

    // Switch to markdown mode so submit goes through onSubmit directly.
    await user.click(screen.getByTestId("toolbar-markdown"));
    // Clear the textarea in case tiptap populated it with HTML on mode switch.
    const textarea = screen.getByTestId("markdown-textarea");
    await user.clear(textarea);
    await user.click(screen.getByTestId("editor-submit-btn"));

    expect(mockCreateMessage).not.toHaveBeenCalled();
  });

  it("sends message and refreshes router on success", async () => {
    mockCreateMessage.mockResolvedValue({});

    const user = userEvent.setup();
    render(<ThreadDetailView {...defaultProps} />);

    await user.click(screen.getByTestId("toolbar-markdown"));
    const textarea = screen.getByTestId("markdown-textarea");
    await user.clear(textarea);
    await user.type(textarea, "Hello world");
    await user.click(screen.getByTestId("editor-submit-btn"));

    await waitFor(() => {
      expect(mockCreateMessage).toHaveBeenCalled();
    });
    expect(mockRefresh).toHaveBeenCalled();
  });

  it("does not fetch uploads when token is unavailable", async () => {
    mockGetToken.mockResolvedValue(null);
    render(<ThreadDetailView {...defaultProps} />);
    // Give effects time to run; fetchThreadUploads should not be called
    // because loadFiles returns early on a null token.
    await waitFor(() => {
      expect(mockFetchThreadUploads).not.toHaveBeenCalled();
    });
  });

  it("uses cached revisions on subsequent revision toggle", async () => {
    const user = userEvent.setup();
    render(<ThreadDetailView {...defaultProps} />);

    // Open, close, then open again — second open uses cached revisions.
    await user.click(screen.getByTestId("revision-toggle"));
    await waitFor(() => expect(screen.getByTestId("no-revisions")).toBeInTheDocument());
    await user.click(screen.getByTestId("revision-toggle"));
    await user.click(screen.getByTestId("revision-toggle"));
    await waitFor(() => expect(screen.getByTestId("no-revisions")).toBeInTheDocument());
    // Only one fetch should have been issued (the second open hits the revisionsLoaded guard).
    expect(mockFetchThreadRevisions).toHaveBeenCalledTimes(1);
  });

  it("closes flag form when cancel is clicked", async () => {
    const user = userEvent.setup();
    render(<ThreadDetailView {...defaultProps} />);

    await user.click(screen.getByTestId("flag-toggle"));
    expect(screen.getByTestId("flag-form")).toBeInTheDocument();

    await user.click(screen.getByTestId("flag-cancel-btn"));
    expect(screen.queryByTestId("flag-form")).not.toBeInTheDocument();
  });
});
