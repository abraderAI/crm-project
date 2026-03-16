import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { OrgRBACEditorWidget } from "./org-rbac-editor-widget";

const mockFetchOrgMembers = vi.fn();
const mockFetchOrgSpaces = vi.fn();
const mockUpdateSpaceRoleOverride = vi.fn();

vi.mock("@/lib/org-api", () => ({
  fetchOrgMembers: (...args: unknown[]) => mockFetchOrgMembers(...args),
  fetchOrgSpaces: (...args: unknown[]) => mockFetchOrgSpaces(...args),
  updateSpaceRoleOverride: (...args: unknown[]) => mockUpdateSpaceRoleOverride(...args),
}));

const MOCK_MEMBERS = [
  { id: "m1", user_id: "alice", role: "admin", org_id: "org-1", created_at: "", updated_at: "" },
  { id: "m2", user_id: "bob", role: "viewer", org_id: "org-1", created_at: "", updated_at: "" },
];

const MOCK_SPACES = [
  {
    id: "s1",
    name: "General",
    slug: "general",
    org_id: "org-1",
    metadata: "",
    type: "general" as const,
    created_at: "",
    updated_at: "",
  },
  {
    id: "s2",
    name: "Support",
    slug: "support",
    org_id: "org-1",
    metadata: "",
    type: "support" as const,
    created_at: "",
    updated_at: "",
  },
];

describe("OrgRBACEditorWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading skeleton initially", () => {
    mockFetchOrgMembers.mockReturnValue(new Promise(() => {}));
    mockFetchOrgSpaces.mockReturnValue(new Promise(() => {}));
    render(<OrgRBACEditorWidget token="token" orgId="org-1" />);
    expect(screen.getByTestId("org-rbac-editor-loading")).toBeInTheDocument();
  });

  it("renders RBAC editor on successful fetch", async () => {
    mockFetchOrgMembers.mockResolvedValue({
      data: MOCK_MEMBERS,
      page_info: { has_more: false },
    });
    mockFetchOrgSpaces.mockResolvedValue({
      data: MOCK_SPACES,
      page_info: { has_more: false },
    });

    render(<OrgRBACEditorWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("org-rbac-editor-widget")).toBeInTheDocument();
    });

    expect(screen.getByTestId("rbac-member-select")).toBeInTheDocument();
    expect(screen.getByText("Space Role Overrides")).toBeInTheDocument();
  });

  it("shows space overrides when member is selected", async () => {
    mockFetchOrgMembers.mockResolvedValue({
      data: MOCK_MEMBERS,
      page_info: { has_more: false },
    });
    mockFetchOrgSpaces.mockResolvedValue({
      data: MOCK_SPACES,
      page_info: { has_more: false },
    });

    render(<OrgRBACEditorWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("rbac-member-select")).toBeInTheDocument();
    });

    fireEvent.change(screen.getByTestId("rbac-member-select"), {
      target: { value: "m1" },
    });

    expect(screen.getByTestId("rbac-space-overrides")).toBeInTheDocument();
    expect(screen.getByTestId("rbac-space-name-s1")).toHaveTextContent("General");
    expect(screen.getByTestId("rbac-space-name-s2")).toHaveTextContent("Support");
  });

  it("calls updateSpaceRoleOverride when role is changed", async () => {
    mockFetchOrgMembers.mockResolvedValue({
      data: MOCK_MEMBERS,
      page_info: { has_more: false },
    });
    mockFetchOrgSpaces.mockResolvedValue({
      data: MOCK_SPACES,
      page_info: { has_more: false },
    });
    mockUpdateSpaceRoleOverride.mockResolvedValue(undefined);

    render(<OrgRBACEditorWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("rbac-member-select")).toBeInTheDocument();
    });

    fireEvent.change(screen.getByTestId("rbac-member-select"), {
      target: { value: "m1" },
    });

    fireEvent.change(screen.getByTestId("rbac-role-select-s1"), {
      target: { value: "moderator" },
    });

    await waitFor(() => {
      expect(mockUpdateSpaceRoleOverride).toHaveBeenCalledWith(
        "token",
        "org-1",
        "m1",
        "s1",
        "moderator",
      );
    });
  });

  it("sends null role for inherit option", async () => {
    mockFetchOrgMembers.mockResolvedValue({
      data: MOCK_MEMBERS,
      page_info: { has_more: false },
    });
    mockFetchOrgSpaces.mockResolvedValue({
      data: MOCK_SPACES,
      page_info: { has_more: false },
    });
    mockUpdateSpaceRoleOverride.mockResolvedValue(undefined);

    render(<OrgRBACEditorWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("rbac-member-select")).toBeInTheDocument();
    });

    fireEvent.change(screen.getByTestId("rbac-member-select"), {
      target: { value: "m1" },
    });

    // First set to something, then back to inherit
    fireEvent.change(screen.getByTestId("rbac-role-select-s1"), {
      target: { value: "inherit" },
    });

    await waitFor(() => {
      expect(mockUpdateSpaceRoleOverride).toHaveBeenCalledWith("token", "org-1", "m1", "s1", null);
    });
  });

  it("shows empty state when no members or spaces", async () => {
    mockFetchOrgMembers.mockResolvedValue({
      data: [],
      page_info: { has_more: false },
    });
    mockFetchOrgSpaces.mockResolvedValue({
      data: [],
      page_info: { has_more: false },
    });

    render(<OrgRBACEditorWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("org-rbac-editor-empty")).toBeInTheDocument();
    });
  });

  it("shows error state on fetch failure", async () => {
    mockFetchOrgMembers.mockRejectedValue(new Error("Network error"));
    mockFetchOrgSpaces.mockRejectedValue(new Error("Network error"));

    render(<OrgRBACEditorWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("org-rbac-editor-error")).toBeInTheDocument();
    });
  });

  it("does not show space overrides until member is selected", async () => {
    mockFetchOrgMembers.mockResolvedValue({
      data: MOCK_MEMBERS,
      page_info: { has_more: false },
    });
    mockFetchOrgSpaces.mockResolvedValue({
      data: MOCK_SPACES,
      page_info: { has_more: false },
    });

    render(<OrgRBACEditorWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("org-rbac-editor-widget")).toBeInTheDocument();
    });

    expect(screen.queryByTestId("rbac-space-overrides")).not.toBeInTheDocument();
  });
});
