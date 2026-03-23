import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ForumHeader } from "./forum-header";

describe("ForumHeader", () => {
  it("renders the forum title and thread count", () => {
    render(<ForumHeader threadCount={42} isAuthenticated={false} />);
    expect(screen.getByText("DEFT General Discussion")).toBeDefined();
    expect(screen.getByText("42 threads")).toBeDefined();
  });

  it("shows singular 'thread' for count of 1", () => {
    render(<ForumHeader threadCount={1} isAuthenticated={false} />);
    expect(screen.getByText("1 thread")).toBeDefined();
  });

  it("links to /forum/new for authenticated users", () => {
    render(<ForumHeader threadCount={0} isAuthenticated={true} />);
    const link = screen.getByTestId("forum-new-thread-btn");
    expect(link.getAttribute("href")).toBe("/forum/new");
  });

  it("links to /sign-in with redirect for unauthenticated users", () => {
    render(<ForumHeader threadCount={0} isAuthenticated={false} />);
    const link = screen.getByTestId("forum-new-thread-btn");
    expect(link.getAttribute("href")).toBe("/sign-in?redirect_url=%2Fforum%2Fnew");
  });
});
