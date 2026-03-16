import { render, screen, act, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { EmbedCodeSnippet } from "./embed-code-snippet";

describe("EmbedCodeSnippet", () => {
  const defaultProps = {
    embedKey: "org_abc123",
  };

  let writeTextMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    // jsdom doesn't provide navigator.clipboard — create a fresh mock for each test
    writeTextMock = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText: writeTextMock },
      writable: true,
      configurable: true,
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders the embed code snippet container", () => {
    render(<EmbedCodeSnippet {...defaultProps} />);
    expect(screen.getByTestId("embed-code-snippet")).toBeInTheDocument();
  });

  it("displays the heading", () => {
    render(<EmbedCodeSnippet {...defaultProps} />);
    expect(screen.getByText("Embed Code")).toBeInTheDocument();
  });

  it("displays the script tag with the correct embed key", () => {
    render(<EmbedCodeSnippet {...defaultProps} />);
    const codeElement = screen.getByTestId("embed-code-text");
    expect(codeElement.textContent).toContain('data-org-key="org_abc123"');
  });

  it("displays a copy button", () => {
    render(<EmbedCodeSnippet {...defaultProps} />);
    expect(screen.getByTestId("embed-copy-btn")).toBeInTheDocument();
  });

  it("calls navigator.clipboard.writeText with the correct snippet on copy click", async () => {
    render(<EmbedCodeSnippet {...defaultProps} />);
    // Use fireEvent instead of userEvent to avoid userEvent replacing navigator.clipboard
    const btn = screen.getByTestId("embed-copy-btn");
    btn.click();
    // Wait for the "Copied!" UI to appear, confirming the clipboard write resolved
    await waitFor(() => {
      expect(screen.getByTestId("embed-copied-msg")).toBeInTheDocument();
    });
    expect(writeTextMock).toHaveBeenCalledTimes(1);
    const copiedText = writeTextMock.mock.calls[0]?.[0] as string;
    expect(copiedText).toContain("<script");
    expect(copiedText).toContain('data-org-key="org_abc123"');
  });

  it("shows 'Copied!' confirmation after clicking copy", async () => {
    const user = userEvent.setup();
    render(<EmbedCodeSnippet {...defaultProps} />);
    await user.click(screen.getByTestId("embed-copy-btn"));
    await waitFor(() => {
      expect(screen.getByTestId("embed-copied-msg")).toHaveTextContent("Copied!");
    });
  });

  it("hides 'Copied!' confirmation after 2 seconds", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(<EmbedCodeSnippet {...defaultProps} />);
    await user.click(screen.getByTestId("embed-copy-btn"));
    await waitFor(() => {
      expect(screen.getByTestId("embed-copied-msg")).toBeInTheDocument();
    });
    await act(async () => {
      vi.advanceTimersByTime(2100);
    });
    expect(screen.queryByTestId("embed-copied-msg")).not.toBeInTheDocument();
    vi.useRealTimers();
  });

  it("updates the embed key reactively", () => {
    const { rerender } = render(<EmbedCodeSnippet embedKey="org_abc123" />);
    rerender(<EmbedCodeSnippet embedKey="org_xyz789" />);
    const codeElement = screen.getByTestId("embed-code-text");
    expect(codeElement.textContent).toContain('data-org-key="org_xyz789"');
  });

  it("renders a code block with the script tag", () => {
    render(<EmbedCodeSnippet {...defaultProps} />);
    const codeBlock = screen.getByTestId("embed-code-text");
    expect(codeBlock.tagName.toLowerCase()).toBe("code");
  });
});
