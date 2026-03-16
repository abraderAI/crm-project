import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { OrgAccessControlWidget } from "./org-access-control-widget";

const mockFetchOrgMembers = vi.fn();
const mockUpdateMemberRole = vi.fn();
const mockRemoveMember = vi.fn();

vi.mock("@/lib/org-api", () => ({
  fetchOrgMembers: (...args: unknown[]) => mockFetchOrgMembers(...args),
  updateMemberRole: (...args: unknown[]) => mockUpdateMemberRole(...args),
  removeMember: (...args: unknown[]) => mockRemoveMember(...args),
}));

const MOCK_MEMBERS = [
  { id: "m1", user_id: "alice", role: "admin", org_id: "org-1", created_at: "", updated_at: "" },
  { id: "m2", user_id: "bob", role: "viewer", org_id: "org-1", created_at: "", updated_at: "" },
  { id: "m3", user_id: "carol", role: "owner", org_id: "org-1", created_at: "", updated_at: "" },
];

describe("OrgAccessControlWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading skeleton initially", () => {
    mockFetchOrgMembers.mockReturnValue(new Promise(() => {}));
    render(<OrgAccessControlWidget token="token" orgId="org-1" />);
    expect(screen.getByTestId("org-access-control-loading")).toBeInTheDocument();
  });

  it("renders member list on successful fetch", async () => {
    mockFetchOrgMembers.mockResolvedValue({
      data: MOCK_MEMBERS,
      page_info: { has_more: false },
    });

    render(<OrgAccessControlWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("org-access-control-widget")).toBeInTheDocument();
    });

    expect(screen.getByTestId("member-name-m1")).toHaveTextContent("alice");
    expect(screen.getByTestId("member-role-m1")).toHaveTextContent("admin");
    expect(screen.getByTestId("member-name-m2")).toHaveTextContent("bob");
    expect(screen.getByTestId("member-role-m2")).toHaveTextContent("viewer");
  });

  it("shows role badges for each member", async () => {
    mockFetchOrgMembers.mockResolvedValue({
      data: MOCK_MEMBERS,
      page_info: { has_more: false },
    });

    render(<OrgAccessControlWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("member-role-m3")).toHaveTextContent("owner");
    });
  });

  it("does not show edit/remove controls for owner role", async () => {
    mockFetchOrgMembers.mockResolvedValue({
      data: MOCK_MEMBERS,
      page_info: { has_more: false },
    });

    render(<OrgAccessControlWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("org-access-control-widget")).toBeInTheDocument();
    });

    // Owner member should not have edit/remove buttons
    expect(screen.queryByTestId("edit-role-m3")).not.toBeInTheDocument();
    expect(screen.queryByTestId("remove-member-m3")).not.toBeInTheDocument();

    // Non-owner members should have controls
    expect(screen.getByTestId("edit-role-m1")).toBeInTheDocument();
    expect(screen.getByTestId("remove-member-m1")).toBeInTheDocument();
  });

  it("shows role selector when edit button is clicked", async () => {
    mockFetchOrgMembers.mockResolvedValue({
      data: MOCK_MEMBERS,
      page_info: { has_more: false },
    });

    render(<OrgAccessControlWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("edit-role-m2")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("edit-role-m2"));
    expect(screen.getByTestId("role-select-m2")).toBeInTheDocument();
  });

  it("calls updateMemberRole when role is changed", async () => {
    mockFetchOrgMembers.mockResolvedValue({
      data: MOCK_MEMBERS,
      page_info: { has_more: false },
    });
    mockUpdateMemberRole.mockResolvedValue({ ...MOCK_MEMBERS[1], role: "contributor" });

    render(<OrgAccessControlWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("edit-role-m2")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("edit-role-m2"));
    fireEvent.change(screen.getByTestId("role-select-m2"), {
      target: { value: "contributor" },
    });

    await waitFor(() => {
      expect(mockUpdateMemberRole).toHaveBeenCalledWith("token", "org-1", "m2", "contributor");
    });
  });

  it("calls removeMember when remove button is clicked", async () => {
    mockFetchOrgMembers.mockResolvedValue({
      data: MOCK_MEMBERS,
      page_info: { has_more: false },
    });
    mockRemoveMember.mockResolvedValue(undefined);

    render(<OrgAccessControlWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("remove-member-m2")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("remove-member-m2"));

    await waitFor(() => {
      expect(mockRemoveMember).toHaveBeenCalledWith("token", "org-1", "m2");
    });
  });

  it("shows empty state when no members", async () => {
    mockFetchOrgMembers.mockResolvedValue({
      data: [],
      page_info: { has_more: false },
    });

    render(<OrgAccessControlWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("org-access-control-empty")).toBeInTheDocument();
    });
  });

  it("shows error state on fetch failure", async () => {
    mockFetchOrgMembers.mockRejectedValue(new Error("Network error"));

    render(<OrgAccessControlWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("org-access-control-error")).toBeInTheDocument();
    });
  });
});
