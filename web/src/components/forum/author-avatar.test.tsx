import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { AuthorAvatar } from "./author-avatar";

describe("AuthorAvatar", () => {
  it("renders '?' for system-seed author", () => {
    render(<AuthorAvatar authorId="system-seed" />);
    expect(screen.getByTestId("author-avatar").textContent).toBe("?");
  });

  it("renders '?' for empty name", () => {
    render(<AuthorAvatar authorId="user-123" />);
    expect(screen.getByTestId("author-avatar").textContent).toBe("?");
  });

  it("renders initials for a two-word name", () => {
    render(<AuthorAvatar authorId="user-456" authorName="Jane Doe" />);
    expect(screen.getByTestId("author-avatar").textContent).toBe("JD");
  });

  it("renders first two chars for a single-word name", () => {
    render(<AuthorAvatar authorId="user-789" authorName="Admin" />);
    expect(screen.getByTestId("author-avatar").textContent).toBe("AD");
  });

  it("renders small size by default", () => {
    render(<AuthorAvatar authorId="test" authorName="Test User" />);
    const el = screen.getByTestId("author-avatar");
    expect(el.className).toContain("h-7");
  });

  it("renders medium size when specified", () => {
    render(<AuthorAvatar authorId="test" authorName="Test User" size="md" />);
    const el = screen.getByTestId("author-avatar");
    expect(el.className).toContain("h-10");
  });

  it("produces deterministic colors for the same ID", () => {
    const { unmount } = render(<AuthorAvatar authorId="stable-id" authorName="A B" />);
    const cls1 = screen.getByTestId("author-avatar").className;
    unmount();
    render(<AuthorAvatar authorId="stable-id" authorName="A B" />);
    const cls2 = screen.getByTestId("author-avatar").className;
    expect(cls1).toBe(cls2);
  });
});
