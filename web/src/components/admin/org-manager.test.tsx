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

import { OrgManager } from "./org-manager";

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
  name: "Suspended Inc",
  slug: "suspended-inc",
  suspended_at: "2026-02-01T00:00:00Z",
  suspend_reason: "Violation",
};

describe("OrgManager", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
  });

  // --- Rendering ---

  it("renders org-manager container", () => {
    render(<OrgManager initialOrgs={[baseOrg]} />);
    expect(screen.getByTestId("org-manager")).toBeInTheDocument();
  });

  it("renders org count in heading", () => {
    render(<OrgManager initialOrgs={[baseOrg]} />);
    expect(screen.getByText("Organizations (1)")).toBeInTheDocument();
  });

  it("renders each org row", () => {
    render(<OrgManager initialOrgs={[baseOrg, suspendedOrg]} />);
    expect(screen.getByTestId("org-row-org_1")).toBeInTheDocument();
    expect(screen.getByTestId("org-row-org_2")).toBeInTheDocument();
  });

  it("shows empty state when no orgs", () => {
    render(<OrgManager initialOrgs={[]} />);
    expect(screen.getByTestId("org-list-empty")).toBeInTheDocument();
  });

  it("renders org name link", () => {
    render(<OrgManager initialOrgs={[baseOrg]} />);
    expect(screen.getByTestId("org-name-link-org_1")).toHaveTextContent("Acme Corp");
  });

  it("renders member count", () => {
    render(<OrgManager initialOrgs={[baseOrg]} />);
    expect(screen.getByTestId("org-member-count-org_1")).toHaveTextContent("5");
  });

  it("shows suspended badge for suspended orgs", () => {
    render(<OrgManager initialOrgs={[suspendedOrg]} />);
    expect(screen.getByTestId("org-suspended-badge-org_2")).toBeInTheDocument();
  });

  it("does not show suspended badge for active orgs", () => {
    render(<OrgManager initialOrgs={[baseOrg]} />);
    expect(screen.queryByTestId("org-suspended-badge-org_1")).not.toBeInTheDocument();
  });

  it("shows Suspend button for active orgs", () => {
    render(<OrgManager initialOrgs={[baseOrg]} />);
    expect(screen.getByTestId("suspend-btn-org_1")).toBeInTheDocument();
  });

  it("shows Unsuspend button for suspended orgs", () => {
    render(<OrgManager initialOrgs={[suspendedOrg]} />);
    expect(screen.getByTestId("unsuspend-btn-org_2")).toBeInTheDocument();
  });

  // --- Create org ---

  it("shows create org button", () => {
    render(<OrgManager initialOrgs={[]} />);
    expect(screen.getByTestId("create-org-btn")).toBeInTheDocument();
  });

  it("toggles create form on button click", async () => {
    const user = userEvent.setup();
    render(<OrgManager initialOrgs={[]} />);

    expect(screen.queryByTestId("create-org-form")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("create-org-btn"));
    expect(screen.getByTestId("create-org-form")).toBeInTheDocument();
  });

  it("hides create form on cancel", async () => {
    const user = userEvent.setup();
    render(<OrgManager initialOrgs={[]} />);

    await user.click(screen.getByTestId("create-org-btn"));
    await user.click(screen.getByTestId("create-org-cancel"));
    expect(screen.queryByTestId("create-org-form")).not.toBeInTheDocument();
  });

  it("disables create submit when name is empty", async () => {
    const user = userEvent.setup();
    render(<OrgManager initialOrgs={[]} />);

    await user.click(screen.getByTestId("create-org-btn"));
    expect(screen.getByTestId("create-org-submit")).toBeDisabled();
  });

  it("enables create submit when name is entered", async () => {
    const user = userEvent.setup();
    render(<OrgManager initialOrgs={[]} />);

    await user.click(screen.getByTestId("create-org-btn"));
    await user.type(screen.getByTestId("create-org-name-input"), "New Org");
    expect(screen.getByTestId("create-org-submit")).toBeEnabled();
  });

  it("calls create endpoint on submit and adds org to list", async () => {
    const user = userEvent.setup();
    const newOrg: AdminOrgDetail = { ...baseOrg, id: "org_new", name: "New Org", slug: "new-org" };
    mockClientMutate.mockResolvedValue(newOrg);
    render(<OrgManager initialOrgs={[]} />);

    await user.click(screen.getByTestId("create-org-btn"));
    await user.type(screen.getByTestId("create-org-name-input"), "New Org");
    await user.click(screen.getByTestId("create-org-submit"));

    expect(mockClientMutate).toHaveBeenCalledWith(
      "POST",
      "/orgs",
      expect.objectContaining({
        token: "test-token",
        body: expect.objectContaining({ name: "New Org" }),
      }),
    );

    await waitFor(() => {
      expect(screen.getByTestId("org-row-org_new")).toBeInTheDocument();
    });
  });

  it("shows success message after create", async () => {
    const user = userEvent.setup();
    const newOrg: AdminOrgDetail = { ...baseOrg, id: "org_new2", name: "Beta Org", slug: "beta" };
    mockClientMutate.mockResolvedValue(newOrg);
    render(<OrgManager initialOrgs={[]} />);

    await user.click(screen.getByTestId("create-org-btn"));
    await user.type(screen.getByTestId("create-org-name-input"), "Beta Org");
    await user.click(screen.getByTestId("create-org-submit"));

    await waitFor(() => {
      expect(screen.getByTestId("org-manager-success")).toBeInTheDocument();
    });
  });

  it("shows error when create fails", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockRejectedValue(new Error("Create failed"));
    render(<OrgManager initialOrgs={[]} />);

    await user.click(screen.getByTestId("create-org-btn"));
    await user.type(screen.getByTestId("create-org-name-input"), "Fail Org");
    await user.click(screen.getByTestId("create-org-submit"));

    await waitFor(() => {
      expect(screen.getByTestId("org-manager-error")).toHaveTextContent("Create failed");
    });
  });

  // --- Suspend ---

  it("opens suspend dialog on Suspend button click", async () => {
    const user = userEvent.setup();
    render(<OrgManager initialOrgs={[baseOrg]} />);

    await user.click(screen.getByTestId("suspend-btn-org_1"));
    expect(screen.getByTestId("suspend-confirm-dialog")).toBeInTheDocument();
  });

  it("closes suspend dialog on cancel", async () => {
    const user = userEvent.setup();
    render(<OrgManager initialOrgs={[baseOrg]} />);

    await user.click(screen.getByTestId("suspend-btn-org_1"));
    await user.click(screen.getByTestId("suspend-cancel-btn"));
    expect(screen.queryByTestId("suspend-confirm-dialog")).not.toBeInTheDocument();
  });

  it("closes suspend dialog on X button", async () => {
    const user = userEvent.setup();
    render(<OrgManager initialOrgs={[baseOrg]} />);

    await user.click(screen.getByTestId("suspend-btn-org_1"));
    await user.click(screen.getByTestId("suspend-dialog-close"));
    expect(screen.queryByTestId("suspend-confirm-dialog")).not.toBeInTheDocument();
  });

  it("calls suspend endpoint and updates badge", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockResolvedValue(undefined);
    render(<OrgManager initialOrgs={[baseOrg]} />);

    await user.click(screen.getByTestId("suspend-btn-org_1"));
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
      expect(screen.getByTestId("org-suspended-badge-org_1")).toBeInTheDocument();
    });
  });

  // --- Unsuspend ---

  it("calls unsuspend endpoint and removes badge", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockResolvedValue(undefined);
    render(<OrgManager initialOrgs={[suspendedOrg]} />);

    expect(screen.getByTestId("org-suspended-badge-org_2")).toBeInTheDocument();
    await user.click(screen.getByTestId("unsuspend-btn-org_2"));

    expect(mockClientMutate).toHaveBeenCalledWith(
      "POST",
      "/admin/orgs/org_2/unsuspend",
      expect.objectContaining({ token: "test-token" }),
    );

    await waitFor(() => {
      expect(screen.queryByTestId("org-suspended-badge-org_2")).not.toBeInTheDocument();
    });
  });

  it("shows error when unsuspend fails", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockRejectedValue(new Error("Unsuspend failed"));
    render(<OrgManager initialOrgs={[suspendedOrg]} />);

    await user.click(screen.getByTestId("unsuspend-btn-org_2"));

    await waitFor(() => {
      expect(screen.getByTestId("org-manager-error")).toHaveTextContent("Unsuspend failed");
    });
  });

  // --- Branch coverage ---

  it("types in description input when create form is open", async () => {
    const user = userEvent.setup();
    render(<OrgManager initialOrgs={[]} />);

    await user.click(screen.getByTestId("create-org-btn"));
    const desc = screen.getByTestId("create-org-desc-input");
    await user.type(desc, "Some description");
    expect(desc).toHaveValue("Some description");
  });

  it("calls router.push when Details link is clicked", async () => {
    const user = userEvent.setup();
    render(<OrgManager initialOrgs={[baseOrg]} />);

    await user.click(screen.getByTestId("org-detail-link-org_1"));
    expect(mockPush).toHaveBeenCalledWith("/admin/orgs/org_1");
  });

  it("shows fallback error when suspend throws non-Error", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockRejectedValue("plain string error");
    render(<OrgManager initialOrgs={[baseOrg]} />);

    await user.click(screen.getByTestId("suspend-btn-org_1"));
    await user.click(screen.getByTestId("suspend-confirm-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("org-manager-error")).toHaveTextContent("Failed to suspend org");
    });
  });

  it("hides create form when Create Org button is toggled off", async () => {
    const user = userEvent.setup();
    render(<OrgManager initialOrgs={[]} />);

    await user.click(screen.getByTestId("create-org-btn"));
    expect(screen.getByTestId("create-org-form")).toBeInTheDocument();
    await user.click(screen.getByTestId("create-org-btn"));
    expect(screen.queryByTestId("create-org-form")).not.toBeInTheDocument();
  });

  it("deletes an org when confirm is accepted", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockResolvedValue(undefined);
    vi.spyOn(window, "confirm").mockReturnValue(true);
    render(<OrgManager initialOrgs={[baseOrg]} />);

    await user.click(screen.getByTestId("delete-org-btn-org_1"));

    await waitFor(() => {
      expect(mockClientMutate).toHaveBeenCalledWith(
        "DELETE",
        "/orgs/acme-corp",
        expect.objectContaining({ token: "test-token" }),
      );
    });

    await waitFor(() => {
      expect(screen.queryByTestId("org-row-org_1")).not.toBeInTheDocument();
    });
  });

  it("does not delete when confirm is cancelled", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "confirm").mockReturnValue(false);
    render(<OrgManager initialOrgs={[baseOrg]} />);

    await user.click(screen.getByTestId("delete-org-btn-org_1"));
    expect(mockClientMutate).not.toHaveBeenCalled();
    expect(screen.getByTestId("org-row-org_1")).toBeInTheDocument();
  });

  it("shows error when delete fails", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockRejectedValue(new Error("Delete failed"));
    vi.spyOn(window, "confirm").mockReturnValue(true);
    render(<OrgManager initialOrgs={[baseOrg]} />);

    await user.click(screen.getByTestId("delete-org-btn-org_1"));

    await waitFor(() => {
      expect(screen.getByTestId("org-manager-error")).toHaveTextContent("Delete failed");
    });
  });
});
