import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import type { UserShadow, OrgMembership } from "@/lib/api-types";

// Mock Clerk auth.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

// Mock next/navigation.
const mockPush = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
}));

// Mock api-client.
const mockClientMutate = vi.fn();
vi.mock("@/lib/api-client", () => ({
  clientMutate: (...args: unknown[]) => mockClientMutate(...args),
}));

import { UserDetail } from "./user-detail";

const baseUser: UserShadow = {
  clerk_user_id: "user_abc123",
  email: "alice@example.com",
  display_name: "Alice Smith",
  avatar_url: "https://img.example.com/alice.png",
  last_seen_at: "2026-03-15T10:00:00Z",
  is_banned: false,
  ban_reason: undefined,
  synced_at: "2026-01-01T00:00:00Z",
  banned_at: null,
  banned_by: undefined,
};

const bannedUser: UserShadow = {
  ...baseUser,
  is_banned: true,
  ban_reason: "Spam",
  banned_at: "2026-03-10T00:00:00Z",
  banned_by: "admin_xyz",
};

const memberships: OrgMembership[] = [
  {
    id: "mem1",
    user_id: "user_abc123",
    org_id: "org1",
    role: "admin",
    created_at: "2026-01-15T00:00:00Z",
    updated_at: "2026-01-15T00:00:00Z",
  },
  {
    id: "mem2",
    user_id: "user_abc123",
    org_id: "org2",
    role: "viewer",
    created_at: "2026-02-01T00:00:00Z",
    updated_at: "2026-02-01T00:00:00Z",
  },
];

describe("UserDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
    sessionStorage.clear();
  });

  afterEach(() => {
    sessionStorage.clear();
  });

  // --- Profile card ---

  it("renders user display name", () => {
    render(<UserDetail user={baseUser} memberships={memberships} />);
    expect(screen.getByTestId("user-display-name")).toHaveTextContent("Alice Smith");
  });

  it("renders user email", () => {
    render(<UserDetail user={baseUser} memberships={memberships} />);
    expect(screen.getByTestId("user-email")).toHaveTextContent("alice@example.com");
  });

  it("renders last seen date", () => {
    render(<UserDetail user={baseUser} memberships={memberships} />);
    expect(screen.getByTestId("user-last-seen")).toBeInTheDocument();
  });

  it("renders joined date from synced_at", () => {
    render(<UserDetail user={baseUser} memberships={memberships} />);
    expect(screen.getByTestId("user-joined")).toBeInTheDocument();
  });

  it("renders avatar when avatar_url is provided", () => {
    render(<UserDetail user={baseUser} memberships={memberships} />);
    const avatar = screen.getByTestId("user-avatar");
    expect(avatar).toBeInTheDocument();
    expect(avatar).toHaveAttribute("src", expect.stringContaining("alice.png"));
  });

  it("renders fallback when no avatar_url", () => {
    const noAvatar = { ...baseUser, avatar_url: undefined };
    render(<UserDetail user={noAvatar} memberships={memberships} />);
    expect(screen.getByTestId("user-avatar-fallback")).toBeInTheDocument();
  });

  // --- Ban status badge ---

  it("does not show banned badge for non-banned user", () => {
    render(<UserDetail user={baseUser} memberships={memberships} />);
    expect(screen.queryByTestId("user-banned-badge")).not.toBeInTheDocument();
  });

  it("shows banned badge for banned user", () => {
    render(<UserDetail user={bannedUser} memberships={memberships} />);
    expect(screen.getByTestId("user-banned-badge")).toBeInTheDocument();
  });

  it("shows ban reason for banned user", () => {
    render(<UserDetail user={bannedUser} memberships={memberships} />);
    expect(screen.getByTestId("user-ban-reason")).toHaveTextContent("Spam");
  });

  // --- Memberships table ---

  it("renders memberships table with correct number of rows", () => {
    render(<UserDetail user={baseUser} memberships={memberships} />);
    expect(screen.getByTestId("membership-row-mem1")).toBeInTheDocument();
    expect(screen.getByTestId("membership-row-mem2")).toBeInTheDocument();
  });

  it("displays org ID and role for each membership", () => {
    render(<UserDetail user={baseUser} memberships={memberships} />);
    expect(screen.getByTestId("membership-org-mem1")).toHaveTextContent("org1");
    expect(screen.getByTestId("membership-role-mem1")).toHaveTextContent("admin");
  });

  it("shows empty state when no memberships", () => {
    render(<UserDetail user={baseUser} memberships={[]} />);
    expect(screen.getByTestId("memberships-empty")).toBeInTheDocument();
  });

  // --- Ban/unban toggle ---

  it("shows ban button for non-banned user", () => {
    render(<UserDetail user={baseUser} memberships={memberships} />);
    expect(screen.getByTestId("ban-toggle-btn")).toHaveTextContent("Ban User");
  });

  it("shows unban button for banned user", () => {
    render(<UserDetail user={bannedUser} memberships={memberships} />);
    expect(screen.getByTestId("ban-toggle-btn")).toHaveTextContent("Unban User");
  });

  it("opens ban confirmation dialog on ban button click", async () => {
    const user = userEvent.setup();
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("ban-toggle-btn"));
    expect(screen.getByTestId("ban-confirm-dialog")).toBeInTheDocument();
  });

  it("allows entering a ban reason", async () => {
    const user = userEvent.setup();
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("ban-toggle-btn"));
    const textarea = screen.getByTestId("ban-reason-input");
    await user.type(textarea, "Spam behavior");
    expect(textarea).toHaveValue("Spam behavior");
  });

  it("calls ban endpoint on confirm", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockResolvedValue({ ...baseUser, is_banned: true });
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("ban-toggle-btn"));
    await user.type(screen.getByTestId("ban-reason-input"), "Spam");
    await user.click(screen.getByTestId("ban-confirm-btn"));

    expect(mockClientMutate).toHaveBeenCalledWith(
      "POST",
      "/admin/users/user_abc123/ban",
      expect.objectContaining({ token: "test-token", body: { reason: "Spam" } }),
    );
  });

  it("calls unban endpoint for banned user on confirm", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockResolvedValue({ ...bannedUser, is_banned: false });
    render(<UserDetail user={bannedUser} memberships={memberships} />);

    await user.click(screen.getByTestId("ban-toggle-btn"));
    await user.click(screen.getByTestId("ban-confirm-btn"));

    expect(mockClientMutate).toHaveBeenCalledWith(
      "POST",
      "/admin/users/user_abc123/unban",
      expect.objectContaining({ token: "test-token" }),
    );
  });

  it("updates ban status after successful ban", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockResolvedValue({ ...baseUser, is_banned: true });
    render(<UserDetail user={baseUser} memberships={memberships} />);

    expect(screen.queryByTestId("user-banned-badge")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("ban-toggle-btn"));
    await user.click(screen.getByTestId("ban-confirm-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("user-banned-badge")).toBeInTheDocument();
    });
  });

  it("closes ban dialog on cancel", async () => {
    const user = userEvent.setup();
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("ban-toggle-btn"));
    expect(screen.getByTestId("ban-confirm-dialog")).toBeInTheDocument();
    await user.click(screen.getByTestId("ban-cancel-btn"));
    expect(screen.queryByTestId("ban-confirm-dialog")).not.toBeInTheDocument();
  });

  // --- GDPR purge ---

  it("shows purge button", () => {
    render(<UserDetail user={baseUser} memberships={memberships} />);
    expect(screen.getByTestId("purge-btn")).toBeInTheDocument();
  });

  it("opens purge dialog on click", async () => {
    const user = userEvent.setup();
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("purge-btn"));
    expect(screen.getByTestId("purge-confirm-dialog")).toBeInTheDocument();
  });

  it("disables purge confirm button when email does not match", async () => {
    const user = userEvent.setup();
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("purge-btn"));
    const input = screen.getByTestId("purge-email-input");
    await user.type(input, "wrong@email.com");
    expect(screen.getByTestId("purge-confirm-btn")).toBeDisabled();
  });

  it("enables purge confirm button when email matches", async () => {
    const user = userEvent.setup();
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("purge-btn"));
    const input = screen.getByTestId("purge-email-input");
    await user.type(input, "alice@example.com");
    expect(screen.getByTestId("purge-confirm-btn")).toBeEnabled();
  });

  it("calls purge endpoint and redirects on success", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockResolvedValue(undefined);
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("purge-btn"));
    await user.type(screen.getByTestId("purge-email-input"), "alice@example.com");
    await user.click(screen.getByTestId("purge-confirm-btn"));

    expect(mockClientMutate).toHaveBeenCalledWith(
      "DELETE",
      "/admin/users/user_abc123/purge",
      expect.objectContaining({ token: "test-token" }),
    );
    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith("/admin/users");
    });
  });

  it("closes purge dialog on cancel", async () => {
    const user = userEvent.setup();
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("purge-btn"));
    expect(screen.getByTestId("purge-confirm-dialog")).toBeInTheDocument();
    await user.click(screen.getByTestId("purge-cancel-btn"));
    expect(screen.queryByTestId("purge-confirm-dialog")).not.toBeInTheDocument();
  });

  // --- Impersonation ---

  it("shows impersonate button", () => {
    render(<UserDetail user={baseUser} memberships={memberships} />);
    expect(screen.getByTestId("impersonate-btn")).toBeInTheDocument();
  });

  it("stores impersonation token in sessionStorage on success", async () => {
    const user = userEvent.setup();
    const expiresAt = new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString();
    mockClientMutate.mockResolvedValue({ token: "imp-token-123", expires_at: expiresAt });
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("impersonate-btn"));

    await waitFor(() => {
      expect(sessionStorage.getItem("impersonation_token")).toBe("imp-token-123");
    });
  });

  it("never stores impersonation token in localStorage", async () => {
    const user = userEvent.setup();
    const expiresAt = new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString();
    mockClientMutate.mockResolvedValue({ token: "imp-token-123", expires_at: expiresAt });
    const localStorageSpy = vi.spyOn(Storage.prototype, "setItem");
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("impersonate-btn"));

    await waitFor(() => {
      expect(sessionStorage.getItem("impersonation_token")).toBe("imp-token-123");
    });

    // sessionStorage and localStorage share the same Storage.prototype in jsdom,
    // so we check specifically that no localStorage call was made.
    expect(localStorage.getItem("impersonation_token")).toBeNull();
    localStorageSpy.mockRestore();
  });

  it("calls impersonate endpoint", async () => {
    const user = userEvent.setup();
    const expiresAt = new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString();
    mockClientMutate.mockResolvedValue({ token: "imp-token-123", expires_at: expiresAt });
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("impersonate-btn"));

    expect(mockClientMutate).toHaveBeenCalledWith(
      "POST",
      "/admin/users/user_abc123/impersonate",
      expect.objectContaining({ token: "test-token" }),
    );
  });

  it("shows active impersonation indicator with clear button", async () => {
    const user = userEvent.setup();
    const expiresAt = new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString();
    mockClientMutate.mockResolvedValue({ token: "imp-token-123", expires_at: expiresAt });
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("impersonate-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("impersonation-active")).toBeInTheDocument();
    });
    expect(screen.getByTestId("impersonation-clear-btn")).toBeInTheDocument();
  });

  it("clears impersonation token on clear button click", async () => {
    const user = userEvent.setup();
    const expiresAt = new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString();
    mockClientMutate.mockResolvedValue({ token: "imp-token-123", expires_at: expiresAt });
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("impersonate-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("impersonation-clear-btn")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("impersonation-clear-btn"));

    expect(sessionStorage.getItem("impersonation_token")).toBeNull();
    expect(screen.queryByTestId("impersonation-active")).not.toBeInTheDocument();
  });

  it("shows countdown timer during impersonation", async () => {
    const user = userEvent.setup();
    const expiresAt = new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString();
    mockClientMutate.mockResolvedValue({ token: "imp-token-123", expires_at: expiresAt });
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("impersonate-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("impersonation-countdown")).toBeInTheDocument();
    });
  });

  // --- Error handling ---

  it("shows error message when ban fails", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockRejectedValue(new Error("Ban failed"));
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("ban-toggle-btn"));
    await user.click(screen.getByTestId("ban-confirm-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("action-error")).toBeInTheDocument();
    });
  });

  it("shows error message when impersonate fails", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockRejectedValue(new Error("Impersonate failed"));
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("impersonate-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("action-error")).toBeInTheDocument();
    });
  });

  // --- Component container ---

  it("renders user-detail container", () => {
    render(<UserDetail user={baseUser} memberships={memberships} />);
    expect(screen.getByTestId("user-detail")).toBeInTheDocument();
  });

  // --- Add to Org ---

  it("shows add-to-org button", () => {
    render(<UserDetail user={baseUser} memberships={memberships} />);
    expect(screen.getByTestId("add-to-org-btn")).toBeInTheDocument();
  });

  it("shows add-to-org form on button click", async () => {
    const user = userEvent.setup();
    render(<UserDetail user={baseUser} memberships={memberships} />);

    expect(screen.queryByTestId("add-to-org-form")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("add-to-org-btn"));
    expect(screen.getByTestId("add-to-org-form")).toBeInTheDocument();
  });

  it("hides add-to-org form when button clicked again", async () => {
    const user = userEvent.setup();
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("add-to-org-btn"));
    await user.click(screen.getByTestId("add-to-org-btn"));
    expect(screen.queryByTestId("add-to-org-form")).not.toBeInTheDocument();
  });

  it("cancels add-to-org form via cancel button", async () => {
    const user = userEvent.setup();
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("add-to-org-btn"));
    await user.click(screen.getByTestId("add-to-org-cancel"));
    expect(screen.queryByTestId("add-to-org-form")).not.toBeInTheDocument();
  });

  it("changes role via add-to-org role select", async () => {
    const user = userEvent.setup();
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("add-to-org-btn"));
    const select = screen.getByTestId("add-to-org-role-select");
    await user.selectOptions(select, "admin");
    expect(select).toHaveValue("admin");
  });

  it("calls POST /orgs/{slug}/members on add-to-org submit", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockResolvedValue(undefined);
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("add-to-org-btn"));
    await user.type(screen.getByTestId("add-to-org-slug-input"), "acme-corp");
    await user.click(screen.getByTestId("add-to-org-submit"));

    expect(mockClientMutate).toHaveBeenCalledWith(
      "POST",
      "/orgs/acme-corp/members",
      expect.objectContaining({
        token: "test-token",
        body: { user_id: "user_abc123", role: "member" },
      }),
    );
  });

  it("shows add-to-org success message after submit", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockResolvedValue(undefined);
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("add-to-org-btn"));
    await user.type(screen.getByTestId("add-to-org-slug-input"), "acme-corp");
    await user.click(screen.getByTestId("add-to-org-submit"));

    await waitFor(() => {
      expect(screen.getByTestId("add-to-org-success")).toBeInTheDocument();
    });
  });

  it("shows error when add-to-org fails", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockRejectedValue(new Error("Org not found"));
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("add-to-org-btn"));
    await user.type(screen.getByTestId("add-to-org-slug-input"), "bad-org");
    await user.click(screen.getByTestId("add-to-org-submit"));

    await waitFor(() => {
      expect(screen.getByTestId("action-error")).toHaveTextContent("Org not found");
    });
  });

  // --- Promote to Platform Admin ---

  it("shows promote-admin button", () => {
    render(<UserDetail user={baseUser} memberships={memberships} />);
    expect(screen.getByTestId("promote-admin-btn")).toBeInTheDocument();
  });

  it("calls POST /admin/platform-admins on promote click", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockResolvedValue(undefined);
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("promote-admin-btn"));

    expect(mockClientMutate).toHaveBeenCalledWith(
      "POST",
      "/admin/platform-admins",
      expect.objectContaining({
        token: "test-token",
        body: { user_id: "user_abc123" },
      }),
    );
  });

  it("shows promote success message after promotion", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockResolvedValue(undefined);
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("promote-admin-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("promote-admin-success")).toBeInTheDocument();
    });
  });

  it("shows error when promote to admin fails", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockRejectedValue(new Error("Promote failed"));
    render(<UserDetail user={baseUser} memberships={memberships} />);

    await user.click(screen.getByTestId("promote-admin-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("action-error")).toHaveTextContent("Promote failed");
    });
  });
});
