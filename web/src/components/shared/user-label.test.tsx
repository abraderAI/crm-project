import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { UserLabel } from "./user-label";

describe("UserLabel", () => {
  it("renders display name when resolved", () => {
    render(
      <UserLabel userId="user-1" resolved={{ display_name: "Alice Smith", org_name: "DEFT" }} />,
    );
    expect(screen.getByTestId("user-label-user-1")).toHaveTextContent("Alice Smith");
  });

  it("renders org badge when org_name is present", () => {
    render(<UserLabel userId="user-1" resolved={{ display_name: "Alice", org_name: "DEFT" }} />);
    expect(screen.getByTestId("user-label-org-user-1")).toHaveTextContent("DEFT");
  });

  it("does not render org badge when org_name is empty", () => {
    render(<UserLabel userId="user-1" resolved={{ display_name: "Alice", org_name: "" }} />);
    expect(screen.queryByTestId("user-label-org-user-1")).not.toBeInTheDocument();
  });

  it("hides org badge when showOrg is false", () => {
    render(
      <UserLabel
        userId="user-1"
        resolved={{ display_name: "Alice", org_name: "DEFT" }}
        showOrg={false}
      />,
    );
    expect(screen.queryByTestId("user-label-org-user-1")).not.toBeInTheDocument();
  });

  it("falls back to truncated userId when not resolved", () => {
    render(<UserLabel userId="user_very_long_clerk_id_12345" />);
    expect(screen.getByTestId("user-label-user_very_long_clerk_id_12345")).toHaveTextContent(
      "user_very_long_c…",
    );
  });

  it("renders short userId as-is when not resolved", () => {
    render(<UserLabel userId="user-short" />);
    expect(screen.getByTestId("user-label-user-short")).toHaveTextContent("user-short");
  });
});
