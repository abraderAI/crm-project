import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { Upload } from "@/lib/api-types";
import { FilePreview, formatFileSize, isImageType } from "./file-preview";

const imageUpload: Upload = {
  id: "up-1",
  org_id: "o-1",
  entity_type: "message",
  entity_id: "m-1",
  filename: "photo.jpg",
  content_type: "image/jpeg",
  size: 1024 * 500,
  storage_path: "/uploads/photo.jpg",
  uploader_id: "u-1",
  created_at: "2025-01-15T10:00:00Z",
  updated_at: "2025-01-15T10:00:00Z",
};

const docUpload: Upload = {
  ...imageUpload,
  id: "up-2",
  filename: "report.pdf",
  content_type: "application/pdf",
  size: 1024 * 1024 * 2.5,
};

describe("isImageType", () => {
  it("returns true for image types", () => {
    expect(isImageType("image/jpeg")).toBe(true);
    expect(isImageType("image/png")).toBe(true);
    expect(isImageType("image/gif")).toBe(true);
  });

  it("returns false for non-image types", () => {
    expect(isImageType("application/pdf")).toBe(false);
    expect(isImageType("text/plain")).toBe(false);
  });
});

describe("formatFileSize", () => {
  it("formats bytes", () => {
    expect(formatFileSize(500)).toBe("500 B");
  });

  it("formats kilobytes", () => {
    expect(formatFileSize(1024 * 5.5)).toBe("5.5 KB");
  });

  it("formats megabytes", () => {
    expect(formatFileSize(1024 * 1024 * 2.5)).toBe("2.5 MB");
  });
});

describe("FilePreview", () => {
  it("renders image preview with thumbnail", () => {
    render(<FilePreview upload={imageUpload} downloadUrl="/api/uploads/up-1" />);
    expect(screen.getByTestId("file-preview-up-1")).toBeInTheDocument();
    expect(screen.getByTestId("file-thumb-up-1")).toBeInTheDocument();
    expect(screen.getByTestId("file-thumb-up-1")).toHaveAttribute("src", "/api/uploads/up-1");
  });

  it("renders document preview with icon", () => {
    render(<FilePreview upload={docUpload} downloadUrl="/api/uploads/up-2" />);
    expect(screen.getByTestId("file-preview-up-2")).toBeInTheDocument();
    expect(screen.getByTestId("file-icon-up-2")).toBeInTheDocument();
    expect(screen.queryByTestId("file-thumb-up-2")).not.toBeInTheDocument();
  });

  it("shows filename", () => {
    render(<FilePreview upload={imageUpload} downloadUrl="/api/uploads/up-1" />);
    expect(screen.getByTestId("file-name-up-1")).toHaveTextContent("photo.jpg");
  });

  it("shows file size", () => {
    render(<FilePreview upload={imageUpload} downloadUrl="/api/uploads/up-1" />);
    expect(screen.getByTestId("file-size-up-1")).toHaveTextContent("500.0 KB");
  });

  it("renders download link", () => {
    render(<FilePreview upload={imageUpload} downloadUrl="/api/uploads/up-1" />);
    const link = screen.getByTestId("file-download-up-1");
    expect(link).toHaveAttribute("href", "/api/uploads/up-1");
    expect(link).toHaveAttribute("download", "photo.jpg");
  });

  it("renders delete button when onDelete provided", () => {
    render(<FilePreview upload={imageUpload} downloadUrl="/api/uploads/up-1" onDelete={vi.fn()} />);
    expect(screen.getByTestId("file-delete-up-1")).toBeInTheDocument();
  });

  it("does not render delete button when onDelete not provided", () => {
    render(<FilePreview upload={imageUpload} downloadUrl="/api/uploads/up-1" />);
    expect(screen.queryByTestId("file-delete-up-1")).not.toBeInTheDocument();
  });

  it("calls onDelete with upload id", async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn();
    render(
      <FilePreview upload={imageUpload} downloadUrl="/api/uploads/up-1" onDelete={onDelete} />,
    );
    await user.click(screen.getByTestId("file-delete-up-1"));
    expect(onDelete).toHaveBeenCalledWith("up-1");
  });

  it("has accessible labels for download and delete", () => {
    render(<FilePreview upload={imageUpload} downloadUrl="/api/uploads/up-1" onDelete={vi.fn()} />);
    expect(screen.getByLabelText("Download photo.jpg")).toBeInTheDocument();
    expect(screen.getByLabelText("Delete photo.jpg")).toBeInTheDocument();
  });
});
