import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";
import type { AdminOrgDetail } from "@/lib/api-types";

// Mock Clerk auth.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

// Mock next/navigation.
const mockPush = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
  notFound: () => null,
}));

// Mock next/link.
vi.mock("next/link", () => ({
  default: ({
    href,
    children,
    ...props
  }: {
    href: string;
    children: React.ReactNode;
    [key: string]: unknown;
  }) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

// Mock api-client.
const mockClientMutate = vi.fn();
vi.mock("@/lib/api-client", () => ({
  clientMutate: (...args: unknown[]) => mockClientMutate(...args),
}));

import { OrgDetailAdmin } from "./org-detail-admin";

const baseOrg: AdminOrgDetail = {
  id: "org_1",
  name: "Acme Corp",
  slug: "acme-corp",
  billing_tier: "pro",
  payment_status: "active",
  description: "Acme description",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
  suspended_at: null,
  suspend_reason: "",
  member_count: 5,
  space_count: 2,
  board_count: 10,
  thread_count: 20,
};

const suspendedOrg: AdminOrgDetail = {
  ...baseOrg,
  id: "org_2",
  name: "Suspended Corp",
  slug: "suspended-corp",
  suspended_at: "2026-02-01T00:00:00Z",
  suspend_reason: "Policy violation",
};

describe("OrgDetailAdmin", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
  });

  // --- Rendering ---

  it("renders org-detail-admin container", () => {
    render(<OrgDetailAdmin org={baseOrg} />);
    expect(screen.getByTestId("org-detail-admin")).toBeInTheDocument();
  });

  it("renders org name", () => {
    render(<OrgDetailAdmin org={baseOrg} />);
    expect(screen.getByTestId("org-detail-name")).toHaveTextContent("Acme Corp");
  });

  it("renders org slug", () => {
    render(<OrgDetailAdmin org={baseOrg} />);
    expect(screen.getByTestId("org-detail-slug")).toHaveTextContent("/acme-corp");
  });

  it("renders stats: member count", () => {
    render(<OrgDetailAdmin org={baseOrg} />);
    expect(screen.getByTestId("stat-members")).toHaveTextContent("5");
  });

  it("renders stats: space count", () => {
    render(<OrgDetailAdmin org={baseOrg} />);
    expect(screen.getByTestId("stat-spaces")).toHaveTextContent("2");
  });

  it("renders stats: board count", () => {
    render(<OrgDetailAdmin org={baseOrg} />);
    expect(screen.getByTestId("stat-boards")).toHaveTextContent("10");
  });

  it("renders stats: thread count", () => {
    render(<OrgDetailAdmin org={baseOrg} />);
    expect(screen.getByTestId("stat-threads")).toHaveTextContent("20");
  });

  it("renders description when present", () => {
    render(<OrgDetailAdmin org={baseOrg} />);
    expect(screen.getByTestId("org-detail-description")).toHaveTextContent("Acme description");
  });

  it("does not render suspended badge for active org", () => {
    render(<OrgDetailAdmin org={baseOrg} />);
    expect(screen.queryByTestId("org-detail-suspended-badge")).not.toBeInTheDocument();
  });

  it("shows suspended badge for suspended org", () => {
    render(<OrgDetailAdmin org={suspendedOrg} />);
    expect(screen.getByTestId("org-detail-suspended-badge")).toBeInTheDocument();
  });

  it("shows Suspend button for active org", () => {
    render(<OrgDetailAdmin org={baseOrg} />);
    expect(screen.getByTestId("suspend-org-btn")).toBeInTheDocument();
  });

  it("shows Unsuspend button for suspended org", () => {
    render(<OrgDetailAdmin org={suspendedOrg} />);
    expect(screen.getByTestId("unsuspend-org-btn")).toBeInTheDocument();
  });

  it("renders back-to-orgs link", () => {
    render(<OrgDetailAdmin org={baseOrg} />);
    const link = screen.getByTestId("back-to-orgs");
    expect(link).toHaveAttribute("href", "/admin/orgs");
  });

  // --- Edit ---

  it("shows edit form on Edit button click", async () => {
    const user = userEvent.setup();
    render(<OrgDetailAdmin org={baseOrg} />);

    expect(screen.queryByTestId("edit-org-form")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("edit-org-btn"));
    expect(screen.getByTestId("edit-org-form")).toBeInTheDocument();
  });

  it("hides edit form on Cancel", async () => {
    const user = userEvent.setup();
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("edit-org-btn"));
    await user.click(screen.getByTestId("edit-org-cancel"));
    expect(screen.queryByTestId("edit-org-form")).not.toBeInTheDocument();
  });

  it("pre-fills edit form with current name", async () => {
    const user = userEvent.setup();
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("edit-org-btn"));
    expect(screen.getByTestId("edit-org-name-input")).toHaveValue("Acme Corp");
  });

  it("disables save when name is empty", async () => {
    const user = userEvent.setup();
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("edit-org-btn"));
    await user.clear(screen.getByTestId("edit-org-name-input"));
    expect(screen.getByTestId("edit-org-save")).toBeDisabled();
  });

  it("calls edit endpoint and updates name", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockResolvedValue({ ...baseOrg, name: "Acme Updated" });
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("edit-org-btn"));
    const nameInput = screen.getByTestId("edit-org-name-input");
    await user.clear(nameInput);
    await user.type(nameInput, "Acme Updated");
    await user.click(screen.getByTestId("edit-org-save"));

    expect(mockClientMutate).toHaveBeenCalledWith(
      "PATCH",
      "/orgs/acme-corp",
      expect.objectContaining({
        token: "test-token",
        body: expect.objectContaining({ name: "Acme Updated" }),
      }),
    );

    await waitFor(() => {
      expect(screen.getByTestId("org-detail-name")).toHaveTextContent("Acme Updated");
    });
  });

  it("shows success message after successful edit", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockResolvedValue({ ...baseOrg, name: "Acme Updated" });
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("edit-org-btn"));
    await user.click(screen.getByTestId("edit-org-save"));

    await waitFor(() => {
      expect(screen.getByTestId("org-detail-success")).toBeInTheDocument();
    });
  });

  it("shows error when edit fails", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockRejectedValue(new Error("Edit failed"));
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("edit-org-btn"));
    await user.click(screen.getByTestId("edit-org-save"));

    await waitFor(() => {
      expect(screen.getByTestId("org-detail-error")).toHaveTextContent("Edit failed");
    });
  });

  // --- Suspend ---

  it("opens suspend dialog on Suspend Org button click", async () => {
    const user = userEvent.setup();
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("suspend-org-btn"));
    expect(screen.getByTestId("suspend-dialog")).toBeInTheDocument();
  });

  it("closes suspend dialog on cancel", async () => {
    const user = userEvent.setup();
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("suspend-org-btn"));
    await user.click(screen.getByTestId("suspend-cancel-btn"));
    expect(screen.queryByTestId("suspend-dialog")).not.toBeInTheDocument();
  });

  it("closes suspend dialog on X button", async () => {
    const user = userEvent.setup();
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("suspend-org-btn"));
    await user.click(screen.getByTestId("suspend-dialog-close"));
    expect(screen.queryByTestId("suspend-dialog")).not.toBeInTheDocument();
  });

  it("calls suspend endpoint and shows suspended badge", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockResolvedValue(undefined);
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("suspend-org-btn"));
    await user.type(screen.getByTestId("suspend-reason-input"), "Abuse");
    await user.click(screen.getByTestId("suspend-confirm-btn"));

    expect(mockClientMutate).toHaveBeenCalledWith(
      "POST",
      "/admin/orgs/org_1/suspend",
      expect.objectContaining({
        token: "test-token",
        body: { reason: "Abuse" },
      }),
    );

    await waitFor(() => {
      expect(screen.getByTestId("org-detail-suspended-badge")).toBeInTheDocument();
    });
  });

  // --- Unsuspend ---

  it("calls unsuspend endpoint and removes badge", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockResolvedValue(undefined);
    render(<OrgDetailAdmin org={suspendedOrg} />);

    await user.click(screen.getByTestId("unsuspend-org-btn"));

    expect(mockClientMutate).toHaveBeenCalledWith(
      "POST",
      "/admin/orgs/org_2/unsuspend",
      expect.objectContaining({ token: "test-token" }),
    );

    await waitFor(() => {
      expect(screen.queryByTestId("org-detail-suspended-badge")).not.toBeInTheDocument();
    });
  });

  // --- Transfer ownership ---

  it("opens transfer dialog on Transfer Ownership button click", async () => {
    const user = userEvent.setup();
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("transfer-ownership-btn"));
    expect(screen.getByTestId("transfer-dialog")).toBeInTheDocument();
  });

  it("closes transfer dialog on cancel", async () => {
    const user = userEvent.setup();
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("transfer-ownership-btn"));
    await user.click(screen.getByTestId("transfer-cancel-btn"));
    expect(screen.queryByTestId("transfer-dialog")).not.toBeInTheDocument();
  });

  it("disables transfer confirm when new owner ID is empty", async () => {
    const user = userEvent.setup();
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("transfer-ownership-btn"));
    expect(screen.getByTestId("transfer-confirm-btn")).toBeDisabled();
  });

  it("calls transfer endpoint and shows success", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockResolvedValue(undefined);
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("transfer-ownership-btn"));
    await user.type(screen.getByTestId("new-owner-input"), "user_newowner");
    await user.click(screen.getByTestId("transfer-confirm-btn"));

    expect(mockClientMutate).toHaveBeenCalledWith(
      "POST",
      "/admin/orgs/org_1/transfer-ownership",
      expect.objectContaining({
        token: "test-token",
        body: { new_owner_user_id: "user_newowner" },
      }),
    );

    await waitFor(() => {
      expect(screen.getByTestId("org-detail-success")).toBeInTheDocument();
    });
  });

  // --- Purge ---

  it("opens purge dialog on GDPR Purge button click", async () => {
    const user = userEvent.setup();
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("purge-org-btn"));
    expect(screen.getByTestId("purge-dialog")).toBeInTheDocument();
  });

  it("closes purge dialog on cancel", async () => {
    const user = userEvent.setup();
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("purge-org-btn"));
    await user.click(screen.getByTestId("purge-cancel-btn"));
    expect(screen.queryByTestId("purge-dialog")).not.toBeInTheDocument();
  });

  it("disables purge confirm when confirmation does not match", async () => {
    const user = userEvent.setup();
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("purge-org-btn"));
    await user.type(screen.getByTestId("purge-confirm-input"), "wrong text");
    expect(screen.getByTestId("purge-confirm-btn")).toBeDisabled();
  });

  it("enables purge confirm when slug confirmation matches", async () => {
    const user = userEvent.setup();
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("purge-org-btn"));
    await user.type(screen.getByTestId("purge-confirm-input"), "purge acme-corp");
    expect(screen.getByTestId("purge-confirm-btn")).toBeEnabled();
  });

  it("calls purge endpoint and redirects", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockResolvedValue(undefined);
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("purge-org-btn"));
    await user.type(screen.getByTestId("purge-confirm-input"), "purge acme-corp");
    await user.click(screen.getByTestId("purge-confirm-btn"));

    expect(mockClientMutate).toHaveBeenCalledWith(
      "DELETE",
      "/admin/orgs/org_1/purge",
      expect.objectContaining({
        token: "test-token",
        body: { confirm: "purge acme-corp" },
      }),
    );

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith("/admin/orgs");
    });
  });

  it("shows error when purge fails", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockRejectedValue(new Error("Purge failed"));
    render(<OrgDetailAdmin org={baseOrg} />);

    await user.click(screen.getByTestId("purge-org-btn"));
    await user.type(screen.getByTestId("purge-confirm-input"), "purge acme-corp");
    await user.click(screen.getByTestId("purge-confirm-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("org-detail-error")).toHaveTextContent("Purge failed");
    });
  });
});
