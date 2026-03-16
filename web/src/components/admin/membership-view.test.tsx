import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";
import type { OrgMembership } from "@/lib/api-types";

// Mock Clerk auth.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

// Mock entity-api membership mutations.
const mockAddMembership = vi.fn();
const mockChangeMembershipRole = vi.fn();
const mockRemoveMembership = vi.fn();
vi.mock("@/lib/entity-api", () => ({
  addMembership: (...args: unknown[]) => mockAddMembership(...args),
  changeMembershipRole: (...args: unknown[]) => mockChangeMembershipRole(...args),
  removeMembership: (...args: unknown[]) => mockRemoveMembership(...args),
}));

import { MembershipView } from "./membership-view";

const member1: OrgMembership = {
  id: "m1",
  user_id: "user-alice",
  role: "admin",
  org_id: "org1",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const member2: OrgMembership = {
  id: "m2",
  user_id: "user-bob",
  role: "viewer",
  org_id: "org1",
  created_at: "2026-01-02T00:00:00Z",
  updated_at: "2026-01-02T00:00:00Z",
};

describe("MembershipView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
    mockAddMembership.mockResolvedValue({
      id: "m-new",
      user_id: "user-dave",
      role: "viewer",
      org_id: "org1",
    });
    mockChangeMembershipRole.mockResolvedValue({ ...member1, role: "moderator" });
    mockRemoveMembership.mockResolvedValue(undefined);
  });

  it("renders MembershipManager with initial members", () => {
    render(<MembershipView initialMembers={[member1, member2]} />);
    expect(screen.getByTestId("membership-manager")).toBeInTheDocument();
    expect(screen.getByTestId("member-item-m1")).toBeInTheDocument();
    expect(screen.getByTestId("member-item-m2")).toBeInTheDocument();
  });

  it("displays Platform as scope label", () => {
    render(<MembershipView initialMembers={[]} />);
    expect(screen.getByText("Platform Members")).toBeInTheDocument();
  });

  it("calls addMembership via entity-api on add", async () => {
    const user = userEvent.setup();
    render(<MembershipView initialMembers={[]} />);

    await user.click(screen.getByTestId("member-add-toggle"));
    await user.type(screen.getByTestId("member-user-input"), "user-dave");
    await user.selectOptions(screen.getByTestId("member-role-select"), "moderator");
    await user.click(screen.getByTestId("member-save-btn"));

    expect(mockAddMembership).toHaveBeenCalledWith("test-token", "user-dave", "moderator");
  });

  it("calls changeMembershipRole via entity-api on role change using user_id", async () => {
    const user = userEvent.setup();
    render(<MembershipView initialMembers={[member1]} />);

    await user.selectOptions(screen.getByTestId("member-role-m1"), "contributor");

    // member1.user_id = "user-alice" (backend uses user_id, not membership id)
    expect(mockChangeMembershipRole).toHaveBeenCalledWith(
      "test-token",
      "user-alice",
      "contributor",
    );
  });

  it("calls removeMembership via entity-api on remove using user_id", async () => {
    const user = userEvent.setup();
    render(<MembershipView initialMembers={[member1]} />);

    await user.click(screen.getByTestId("member-remove-m1"));

    // member1.user_id = "user-alice" (backend uses user_id, not membership id)
    expect(mockRemoveMembership).toHaveBeenCalledWith("test-token", "user-alice");
  });
});
