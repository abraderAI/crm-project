import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";
import type { Flag } from "@/lib/api-types";

// Mock Clerk auth.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

// Mock entity-api flag mutations.
const mockResolveFlag = vi.fn();
const mockDismissFlag = vi.fn();
vi.mock("@/lib/entity-api", () => ({
  resolveFlag: (...args: unknown[]) => mockResolveFlag(...args),
  dismissFlag: (...args: unknown[]) => mockDismissFlag(...args),
}));

import { ModerationView } from "./moderation-view";

const pendingFlag: Flag = {
  id: "f1",
  thread_id: "t1",
  reporter_id: "u1",
  reason: "Spam or misleading",
  status: "pending",
  created_at: "2026-01-15T00:00:00Z",
  updated_at: "2026-01-15T00:00:00Z",
};

const resolvedFlag: Flag = {
  id: "f2",
  thread_id: "t2",
  reporter_id: "u2",
  reason: "Off-topic content",
  status: "resolved",
  resolution_note: "Addressed",
  created_at: "2026-01-10T00:00:00Z",
  updated_at: "2026-01-12T00:00:00Z",
};

describe("ModerationView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
    mockResolveFlag.mockResolvedValue({ ...pendingFlag, status: "resolved" });
    mockDismissFlag.mockResolvedValue({ ...pendingFlag, status: "dismissed" });
  });

  it("renders ModerationQueue with initial flags", () => {
    render(<ModerationView initialFlags={[pendingFlag, resolvedFlag]} />);
    expect(screen.getByTestId("moderation-queue")).toBeInTheDocument();
    expect(screen.getByTestId("flag-item-f1")).toBeInTheDocument();
    expect(screen.getByTestId("flag-item-f2")).toBeInTheDocument();
  });

  it("calls resolveFlag via entity-api on resolve", async () => {
    const user = userEvent.setup();
    render(<ModerationView initialFlags={[pendingFlag]} />);

    await user.click(screen.getByTestId("flag-resolve-f1"));

    expect(mockResolveFlag).toHaveBeenCalledWith("test-token", "f1", "");
  });

  it("calls dismissFlag via entity-api on dismiss", async () => {
    const user = userEvent.setup();
    render(<ModerationView initialFlags={[pendingFlag]} />);

    await user.click(screen.getByTestId("flag-dismiss-f1"));

    expect(mockDismissFlag).toHaveBeenCalledWith("test-token", "f1");
  });

  it("shows empty state when no flags", () => {
    render(<ModerationView initialFlags={[]} />);
    expect(screen.getByTestId("moderation-empty")).toBeInTheDocument();
  });
});
