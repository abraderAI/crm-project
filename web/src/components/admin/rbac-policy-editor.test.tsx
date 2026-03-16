import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";
import type { EffectivePolicy } from "@/lib/api-types";

// Mock Clerk auth.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

// Mock api-client.
const mockClientMutate = vi.fn();
vi.mock("@/lib/api-client", () => ({
  clientMutate: (...args: unknown[]) => mockClientMutate(...args),
}));

import { RBACPolicyEditor } from "./rbac-policy-editor";

const RESOLUTION_STRATEGIES = ["highest_role", "lowest_role", "most_specific"];

const basePolicy: EffectivePolicy = {
  resolution: {
    strategy: "highest_role",
    order: ["org", "space", "board"],
  },
  roles: {
    hierarchy: ["viewer", "commenter", "contributor", "moderator", "admin", "owner"],
    permissions: {
      viewer: ["read"],
      commenter: ["read", "comment"],
      contributor: ["read", "comment", "create"],
      moderator: ["read", "comment", "create", "moderate"],
      admin: ["read", "comment", "create", "moderate", "admin"],
      owner: ["read", "comment", "create", "moderate", "admin", "owner"],
    },
  },
  defaults: {
    org_member_role: "viewer",
    space_member_role: "viewer",
    board_member_role: "viewer",
  },
};

describe("RBACPolicyEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
    mockClientMutate.mockResolvedValue(basePolicy);
  });

  it("renders the editor heading", () => {
    render(<RBACPolicyEditor policy={basePolicy} />);
    expect(screen.getByText("RBAC Policy Editor")).toBeInTheDocument();
  });

  it("renders the resolution strategy dropdown", () => {
    render(<RBACPolicyEditor policy={basePolicy} />);
    const select = screen.getByTestId("strategy-select") as HTMLSelectElement;
    expect(select).toBeInTheDocument();
    expect(select.value).toBe("highest_role");
  });

  it("renders all strategy options", () => {
    render(<RBACPolicyEditor policy={basePolicy} />);
    const select = screen.getByTestId("strategy-select") as HTMLSelectElement;
    const options = Array.from(select.options).map((o) => o.value);
    for (const s of RESOLUTION_STRATEGIES) {
      expect(options).toContain(s);
    }
  });

  it("renders resolution order display", () => {
    render(<RBACPolicyEditor policy={basePolicy} />);
    expect(screen.getByTestId("resolution-order")).toBeInTheDocument();
    expect(screen.getByText("org → space → board")).toBeInTheDocument();
  });

  it("renders the role hierarchy list", () => {
    render(<RBACPolicyEditor policy={basePolicy} />);
    expect(screen.getByTestId("role-hierarchy")).toBeInTheDocument();
    for (const role of basePolicy.roles.hierarchy) {
      expect(screen.getByTestId(`hierarchy-role-${role}`)).toBeInTheDocument();
    }
  });

  it("renders defaults section with three role selectors", () => {
    render(<RBACPolicyEditor policy={basePolicy} />);
    expect(screen.getByTestId("default-org-role")).toBeInTheDocument();
    expect(screen.getByTestId("default-space-role")).toBeInTheDocument();
    expect(screen.getByTestId("default-board-role")).toBeInTheDocument();
  });

  it("defaults selectors show current values", () => {
    render(<RBACPolicyEditor policy={basePolicy} />);
    const orgSelect = screen.getByTestId("default-org-role") as HTMLSelectElement;
    expect(orgSelect.value).toBe("viewer");
  });

  it("renders save button", () => {
    render(<RBACPolicyEditor policy={basePolicy} />);
    expect(screen.getByTestId("policy-save-btn")).toBeInTheDocument();
  });

  it("calls PATCH /v1/admin/rbac-policy on save", async () => {
    const user = userEvent.setup();
    render(<RBACPolicyEditor policy={basePolicy} />);

    // Change org default to contributor
    await user.selectOptions(screen.getByTestId("default-org-role"), "contributor");
    await user.click(screen.getByTestId("policy-save-btn"));

    await waitFor(() => {
      expect(mockClientMutate).toHaveBeenCalledWith(
        "PATCH",
        "/admin/rbac-policy",
        expect.objectContaining({
          token: "test-token",
          body: expect.objectContaining({
            defaults: expect.objectContaining({
              org_member_role: "contributor",
            }),
          }),
        }),
      );
    });
  });

  it("shows success message after save", async () => {
    const user = userEvent.setup();
    render(<RBACPolicyEditor policy={basePolicy} />);

    await user.click(screen.getByTestId("policy-save-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("save-success")).toBeInTheDocument();
    });
  });

  it("shows error message on save failure", async () => {
    mockClientMutate.mockRejectedValue(new Error("Failed"));
    const user = userEvent.setup();
    render(<RBACPolicyEditor policy={basePolicy} />);

    await user.click(screen.getByTestId("policy-save-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("save-error")).toBeInTheDocument();
    });
  });

  it("disables save button while saving", async () => {
    let resolvePromise: (v: EffectivePolicy) => void;
    mockClientMutate.mockReturnValue(
      new Promise<EffectivePolicy>((resolve) => {
        resolvePromise = resolve;
      }),
    );

    const user = userEvent.setup();
    render(<RBACPolicyEditor policy={basePolicy} />);

    await user.click(screen.getByTestId("policy-save-btn"));
    expect(screen.getByTestId("policy-save-btn")).toBeDisabled();

    resolvePromise!(basePolicy);
    await waitFor(() => {
      expect(screen.getByTestId("policy-save-btn")).not.toBeDisabled();
    });
  });

  it("updates strategy on dropdown change", async () => {
    const user = userEvent.setup();
    render(<RBACPolicyEditor policy={basePolicy} />);

    await user.selectOptions(screen.getByTestId("strategy-select"), "most_specific");
    const select = screen.getByTestId("strategy-select") as HTMLSelectElement;
    expect(select.value).toBe("most_specific");
  });

  it("renders permissions for each role in hierarchy", () => {
    render(<RBACPolicyEditor policy={basePolicy} />);
    expect(screen.getByTestId("role-permissions-viewer")).toBeInTheDocument();
    expect(screen.getByTestId("role-permissions-owner")).toBeInTheDocument();
  });
});
