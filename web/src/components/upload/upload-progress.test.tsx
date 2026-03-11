import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { UploadProgress, formatProgress, type UploadProgressItem } from "./upload-progress";

describe("formatProgress", () => {
  it("formats progress percentage", () => {
    expect(formatProgress(50)).toBe("50%");
  });

  it("clamps to 0%", () => {
    expect(formatProgress(-10)).toBe("0%");
  });

  it("clamps to 100%", () => {
    expect(formatProgress(150)).toBe("100%");
  });

  it("rounds to nearest integer", () => {
    expect(formatProgress(33.7)).toBe("34%");
  });
});

describe("UploadProgress", () => {
  const items: UploadProgressItem[] = [
    { id: "u-1", filename: "photo.jpg", progress: 45 },
    { id: "u-2", filename: "doc.pdf", progress: 100 },
  ];

  it("returns null when no items", () => {
    const { container } = render(<UploadProgress items={[]} />);
    expect(container.firstChild).toBeNull();
  });

  it("renders progress container", () => {
    render(<UploadProgress items={items} />);
    expect(screen.getByTestId("upload-progress")).toBeInTheDocument();
  });

  it("renders all items", () => {
    render(<UploadProgress items={items} />);
    expect(screen.getByTestId("upload-item-u-1")).toBeInTheDocument();
    expect(screen.getByTestId("upload-item-u-2")).toBeInTheDocument();
  });

  it("shows filename", () => {
    render(<UploadProgress items={items} />);
    expect(screen.getByTestId("upload-filename-u-1")).toHaveTextContent("photo.jpg");
  });

  it("shows progress percentage", () => {
    render(<UploadProgress items={items} />);
    expect(screen.getByTestId("upload-percent-u-1")).toHaveTextContent("45%");
    expect(screen.getByTestId("upload-percent-u-2")).toHaveTextContent("100%");
  });

  it("renders progress bar", () => {
    render(<UploadProgress items={items} />);
    const bar = screen.getByTestId("upload-bar-u-1");
    expect(bar).toHaveStyle({ width: "45%" });
  });

  it("shows cancel button for in-progress uploads", () => {
    render(<UploadProgress items={items} onCancel={vi.fn()} />);
    expect(screen.getByTestId("upload-cancel-u-1")).toBeInTheDocument();
  });

  it("hides cancel button for completed uploads", () => {
    render(<UploadProgress items={items} onCancel={vi.fn()} />);
    expect(screen.queryByTestId("upload-cancel-u-2")).not.toBeInTheDocument();
  });

  it("calls onCancel with item id", async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    render(<UploadProgress items={items} onCancel={onCancel} />);

    await user.click(screen.getByTestId("upload-cancel-u-1"));
    expect(onCancel).toHaveBeenCalledWith("u-1");
  });

  it("shows error state", () => {
    const errorItems: UploadProgressItem[] = [
      { id: "u-3", filename: "bad.exe", progress: 0, error: "Type not allowed" },
    ];
    render(<UploadProgress items={errorItems} />);
    expect(screen.getByTestId("upload-percent-u-3")).toHaveTextContent("Failed");
    expect(screen.getByTestId("upload-error-u-3")).toHaveTextContent("Type not allowed");
  });

  it("hides progress bar when error", () => {
    const errorItems: UploadProgressItem[] = [
      { id: "u-3", filename: "bad.exe", progress: 0, error: "Failed" },
    ];
    render(<UploadProgress items={errorItems} />);
    expect(screen.queryByTestId("upload-bar-u-3")).not.toBeInTheDocument();
  });

  it("hides cancel button when error", () => {
    const errorItems: UploadProgressItem[] = [
      { id: "u-3", filename: "bad.exe", progress: 0, error: "Failed" },
    ];
    render(<UploadProgress items={errorItems} onCancel={vi.fn()} />);
    expect(screen.queryByTestId("upload-cancel-u-3")).not.toBeInTheDocument();
  });
});
