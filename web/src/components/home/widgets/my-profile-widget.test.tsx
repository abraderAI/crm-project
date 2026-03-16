import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MyProfileWidget, type ProfileData } from "./my-profile-widget";

vi.mock("next/link", () => ({
  default: ({
    children,
    href,
    ...rest
  }: {
    children: React.ReactNode;
    href: string;
    className?: string;
    "data-testid"?: string;
  }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}));

vi.mock("next/image", () => ({
  default: (props: Record<string, unknown>) => {
    // eslint-disable-next-line @next/next/no-img-element, jsx-a11y/alt-text
    return <img {...props} />;
  },
}));

const MOCK_PROFILE: ProfileData = {
  displayName: "Jane Developer",
  email: "jane@example.com",
  avatarUrl: "https://example.com/avatar.jpg",
  accountStatus: "Active",
};

describe("MyProfileWidget", () => {
  it("shows loading skeleton when isLoading", () => {
    render(<MyProfileWidget profile={null} isLoading={true} />);
    expect(screen.getByTestId("my-profile-loading")).toBeInTheDocument();
  });

  it("shows empty state when profile is null", () => {
    render(<MyProfileWidget profile={null} />);
    expect(screen.getByTestId("my-profile-empty")).toBeInTheDocument();
    expect(screen.getByText("Profile information unavailable.")).toBeInTheDocument();
  });

  it("renders profile data correctly", () => {
    render(<MyProfileWidget profile={MOCK_PROFILE} />);
    expect(screen.getByTestId("my-profile-widget")).toBeInTheDocument();
    expect(screen.getByTestId("my-profile-name")).toHaveTextContent("Jane Developer");
    expect(screen.getByTestId("my-profile-email")).toHaveTextContent("jane@example.com");
    expect(screen.getByTestId("my-profile-status")).toHaveTextContent("Active");
  });

  it("renders avatar image when avatarUrl is provided", () => {
    render(<MyProfileWidget profile={MOCK_PROFILE} />);
    expect(screen.getByTestId("my-profile-avatar")).toBeInTheDocument();
    expect(screen.getByTestId("my-profile-avatar")).toHaveAttribute(
      "src",
      "https://example.com/avatar.jpg",
    );
  });

  it("renders avatar placeholder when no avatarUrl", () => {
    const profile = { ...MOCK_PROFILE, avatarUrl: null };
    render(<MyProfileWidget profile={profile} />);
    expect(screen.getByTestId("my-profile-avatar-placeholder")).toBeInTheDocument();
    expect(screen.queryByTestId("my-profile-avatar")).not.toBeInTheDocument();
  });

  it("defaults accountStatus to Active when not provided", () => {
    const profile = { displayName: "Test", email: "test@test.com" };
    render(<MyProfileWidget profile={profile} />);
    expect(screen.getByTestId("my-profile-status")).toHaveTextContent("Active");
  });

  it("renders edit profile link", () => {
    render(<MyProfileWidget profile={MOCK_PROFILE} />);
    const link = screen.getByTestId("my-profile-edit-link");
    expect(link).toHaveAttribute("href", "/settings/profile");
    expect(link).toHaveTextContent("Edit profile");
  });
});
