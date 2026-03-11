import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { MembershipManager, type MembershipItem } from "./membership-manager";

const members: MembershipItem[] = [
  { id: "m1", user_id: "user-alice", role: "admin" },
  { id: "m2", user_id: "user-bob", role: "viewer" },
  { id: "m3", user_id: "user-carol", role: "moderator" },
];

const defaultProps = {
  members,
  onAdd: vi.fn(),
  onChangeRole: vi.fn(),
  onRemove: vi.fn(),
};

describe("MembershipManager", () => {
  it("renders the heading with scope label", () => {
    render(<MembershipManager {...defaultProps} scopeLabel="Space" />);
    expect(screen.getByText("Space Members")).toBeInTheDocument();
  });

  it("renders default scope label", () => {
    render(<MembershipManager {...defaultProps} />);
    expect(screen.getByText("Organization Members")).toBeInTheDocument();
  });

  it("renders the membership icon", () => {
    render(<MembershipManager {...defaultProps} />);
    expect(screen.getByTestId("membership-icon")).toBeInTheDocument();
  });

  it("shows member count", () => {
    render(<MembershipManager {...defaultProps} />);
    expect(screen.getByTestId("member-count")).toHaveTextContent("3");
  });

  it("shows empty state", () => {
    render(<MembershipManager {...defaultProps} members={[]} />);
    expect(screen.getByTestId("membership-empty")).toHaveTextContent("No members.");
  });

  it("shows loading state", () => {
    render(<MembershipManager {...defaultProps} members={[]} loading={true} />);
    expect(screen.getByTestId("membership-loading")).toBeInTheDocument();
  });

  it("renders member items", () => {
    render(<MembershipManager {...defaultProps} />);
    expect(screen.getByTestId("member-item-m1")).toBeInTheDocument();
    expect(screen.getByTestId("member-item-m2")).toBeInTheDocument();
  });

  it("displays user IDs", () => {
    render(<MembershipManager {...defaultProps} />);
    expect(screen.getByTestId("member-user-m1")).toHaveTextContent("user-alice");
  });

  it("displays role selects with current values", () => {
    render(<MembershipManager {...defaultProps} />);
    const roleSelect = screen.getByTestId("member-role-m1") as HTMLSelectElement;
    expect(roleSelect.value).toBe("admin");
  });

  it("calls onChangeRole when role changed", async () => {
    const user = userEvent.setup();
    const onChangeRole = vi.fn();
    render(<MembershipManager {...defaultProps} onChangeRole={onChangeRole} />);

    await user.selectOptions(screen.getByTestId("member-role-m2"), "contributor");
    expect(onChangeRole).toHaveBeenCalledWith("m2", "contributor");
  });

  it("calls onRemove when remove clicked", async () => {
    const user = userEvent.setup();
    const onRemove = vi.fn();
    render(<MembershipManager {...defaultProps} onRemove={onRemove} />);

    await user.click(screen.getByTestId("member-remove-m1"));
    expect(onRemove).toHaveBeenCalledWith("m1");
  });

  it("shows add form when Add Member clicked", async () => {
    const user = userEvent.setup();
    render(<MembershipManager {...defaultProps} />);

    expect(screen.queryByTestId("member-add-form")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("member-add-toggle"));
    expect(screen.getByTestId("member-add-form")).toBeInTheDocument();
  });

  it("adds member with user ID and role", async () => {
    const user = userEvent.setup();
    const onAdd = vi.fn();
    render(<MembershipManager {...defaultProps} onAdd={onAdd} />);

    await user.click(screen.getByTestId("member-add-toggle"));
    await user.type(screen.getByTestId("member-user-input"), "user-dave");
    await user.selectOptions(screen.getByTestId("member-role-select"), "moderator");
    await user.click(screen.getByTestId("member-save-btn"));
    expect(onAdd).toHaveBeenCalledWith("user-dave", "moderator");
  });

  it("shows error for empty user ID", async () => {
    const user = userEvent.setup();
    render(<MembershipManager {...defaultProps} />);

    await user.click(screen.getByTestId("member-add-toggle"));
    await user.click(screen.getByTestId("member-save-btn"));
    expect(screen.getByTestId("member-error")).toHaveTextContent("User ID is required.");
  });

  it("hides add form after successful add", async () => {
    const user = userEvent.setup();
    render(<MembershipManager {...defaultProps} />);

    await user.click(screen.getByTestId("member-add-toggle"));
    await user.type(screen.getByTestId("member-user-input"), "user-dave");
    await user.click(screen.getByTestId("member-save-btn"));
    expect(screen.queryByTestId("member-add-form")).not.toBeInTheDocument();
  });

  it("hides add form on cancel", async () => {
    const user = userEvent.setup();
    render(<MembershipManager {...defaultProps} />);

    await user.click(screen.getByTestId("member-add-toggle"));
    await user.click(screen.getByTestId("member-cancel-btn"));
    expect(screen.queryByTestId("member-add-form")).not.toBeInTheDocument();
  });

  it("renders all role options in select", () => {
    render(<MembershipManager {...defaultProps} />);
    const select = screen.getByTestId("member-role-m1") as HTMLSelectElement;
    expect(select.options).toHaveLength(6);
  });

  it("renders the member list container", () => {
    render(<MembershipManager {...defaultProps} />);
    expect(screen.getByTestId("member-list")).toBeInTheDocument();
  });
});
