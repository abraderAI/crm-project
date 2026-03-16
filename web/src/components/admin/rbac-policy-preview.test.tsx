import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";

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

import { RBACPolicyPreview } from "./rbac-policy-preview";

const previewResponse = {
  user_id: "user-123",
  entity_type: "org",
  entity_id: "org-456",
  role: "admin",
  permissions: ["read", "comment", "create", "moderate", "admin"],
};

describe("RBACPolicyPreview", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
    mockClientMutate.mockResolvedValue(previewResponse);
  });

  it("renders the preview heading", () => {
    render(<RBACPolicyPreview />);
    expect(screen.getByText("Dry-Run Role Preview")).toBeInTheDocument();
  });

  it("renders user ID input", () => {
    render(<RBACPolicyPreview />);
    expect(screen.getByTestId("preview-user-id")).toBeInTheDocument();
  });

  it("renders entity type dropdown with org/space/board options", () => {
    render(<RBACPolicyPreview />);
    const select = screen.getByTestId("preview-entity-type") as HTMLSelectElement;
    expect(select).toBeInTheDocument();
    const options = Array.from(select.options).map((o) => o.value);
    expect(options).toContain("org");
    expect(options).toContain("space");
    expect(options).toContain("board");
  });

  it("renders entity ID input", () => {
    render(<RBACPolicyPreview />);
    expect(screen.getByTestId("preview-entity-id")).toBeInTheDocument();
  });

  it("renders submit button", () => {
    render(<RBACPolicyPreview />);
    expect(screen.getByTestId("preview-submit-btn")).toBeInTheDocument();
  });

  it("calls POST /v1/admin/rbac-policy/preview on submit", async () => {
    const user = userEvent.setup();
    render(<RBACPolicyPreview />);

    await user.type(screen.getByTestId("preview-user-id"), "user-123");
    await user.selectOptions(screen.getByTestId("preview-entity-type"), "org");
    await user.type(screen.getByTestId("preview-entity-id"), "org-456");
    await user.click(screen.getByTestId("preview-submit-btn"));

    await waitFor(() => {
      expect(mockClientMutate).toHaveBeenCalledWith("POST", "/admin/rbac-policy/preview", {
        token: "test-token",
        body: {
          user_id: "user-123",
          entity_type: "org",
          entity_id: "org-456",
        },
      });
    });
  });

  it("displays resolved role after successful preview", async () => {
    const user = userEvent.setup();
    render(<RBACPolicyPreview />);

    await user.type(screen.getByTestId("preview-user-id"), "user-123");
    await user.type(screen.getByTestId("preview-entity-id"), "org-456");
    await user.click(screen.getByTestId("preview-submit-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("preview-result")).toBeInTheDocument();
      expect(screen.getByTestId("preview-resolved-role")).toHaveTextContent("admin");
    });
  });

  it("displays resolved permissions after successful preview", async () => {
    const user = userEvent.setup();
    render(<RBACPolicyPreview />);

    await user.type(screen.getByTestId("preview-user-id"), "user-123");
    await user.type(screen.getByTestId("preview-entity-id"), "org-456");
    await user.click(screen.getByTestId("preview-submit-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("preview-permissions")).toBeInTheDocument();
    });
  });

  it("shows error message on preview failure", async () => {
    mockClientMutate.mockRejectedValue(new Error("Preview failed"));
    const user = userEvent.setup();
    render(<RBACPolicyPreview />);

    await user.type(screen.getByTestId("preview-user-id"), "user-123");
    await user.type(screen.getByTestId("preview-entity-id"), "org-456");
    await user.click(screen.getByTestId("preview-submit-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("preview-error")).toBeInTheDocument();
    });
  });

  it("shows validation error when user ID is empty", async () => {
    const user = userEvent.setup();
    render(<RBACPolicyPreview />);

    await user.click(screen.getByTestId("preview-submit-btn"));

    expect(screen.getByTestId("preview-validation-error")).toBeInTheDocument();
    expect(mockClientMutate).not.toHaveBeenCalled();
  });

  it("shows validation error when entity ID is empty", async () => {
    const user = userEvent.setup();
    render(<RBACPolicyPreview />);

    await user.type(screen.getByTestId("preview-user-id"), "user-123");
    await user.click(screen.getByTestId("preview-submit-btn"));

    expect(screen.getByTestId("preview-validation-error")).toBeInTheDocument();
    expect(mockClientMutate).not.toHaveBeenCalled();
  });

  it("disables submit button while loading", async () => {
    let resolvePromise: (v: unknown) => void;
    mockClientMutate.mockReturnValue(
      new Promise((resolve) => {
        resolvePromise = resolve;
      }),
    );

    const user = userEvent.setup();
    render(<RBACPolicyPreview />);

    await user.type(screen.getByTestId("preview-user-id"), "user-123");
    await user.type(screen.getByTestId("preview-entity-id"), "org-456");
    await user.click(screen.getByTestId("preview-submit-btn"));

    expect(screen.getByTestId("preview-submit-btn")).toBeDisabled();

    resolvePromise!(previewResponse);
    await waitFor(() => {
      expect(screen.getByTestId("preview-submit-btn")).not.toBeDisabled();
    });
  });

  it("displays entity type in result", async () => {
    const user = userEvent.setup();
    render(<RBACPolicyPreview />);

    await user.type(screen.getByTestId("preview-user-id"), "user-123");
    await user.selectOptions(screen.getByTestId("preview-entity-type"), "space");
    await user.type(screen.getByTestId("preview-entity-id"), "space-789");
    await user.click(screen.getByTestId("preview-submit-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("preview-result")).toBeInTheDocument();
    });
  });

  it("does not render result before submission", () => {
    render(<RBACPolicyPreview />);
    expect(screen.queryByTestId("preview-result")).not.toBeInTheDocument();
  });
});
