import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { GetStartedWidget } from "./get-started-widget";

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

describe("GetStartedWidget", () => {
  it("renders the widget container", () => {
    render(<GetStartedWidget />);
    expect(screen.getByTestId("get-started-widget")).toBeInTheDocument();
  });

  it("displays community join message", () => {
    render(<GetStartedWidget />);
    expect(screen.getByText("Join the DEFT community today")).toBeInTheDocument();
  });

  it("lists feature highlights", () => {
    render(<GetStartedWidget />);
    const features = screen.getByTestId("get-started-features");
    expect(features.querySelectorAll("li")).toHaveLength(3);
    expect(screen.getByText("Access community forums and documentation")).toBeInTheDocument();
    expect(screen.getByText("Create and track support tickets")).toBeInTheDocument();
    expect(screen.getByText("Connect with other developers")).toBeInTheDocument();
  });

  it("renders sign-up CTA link pointing to /sign-up", () => {
    render(<GetStartedWidget />);
    const cta = screen.getByTestId("get-started-cta");
    expect(cta).toHaveAttribute("href", "/sign-up");
    expect(cta).toHaveTextContent("Sign up for free");
  });
});
