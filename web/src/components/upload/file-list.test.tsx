import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { FileList, type FileItem } from "./file-list";

const mockFiles: FileItem[] = [
  {
    id: "f1",
    filename: "report.pdf",
    contentType: "application/pdf",
    size: 1048576,
    downloadUrl: "/api/uploads/f1",
  },
  {
    id: "f2",
    filename: "photo.png",
    contentType: "image/png",
    size: 2048,
    downloadUrl: "/api/uploads/f2",
  },
  {
    id: "f3",
    filename: "data.json",
    contentType: "application/json",
    size: 512,
    downloadUrl: "/api/uploads/f3",
  },
];

describe("FileList", () => {
  it("renders no files message when empty", () => {
    render(<FileList files={[]} />);
    expect(screen.getByTestId("no-files")).toHaveTextContent("No files uploaded.");
  });

  it("does not render list when empty", () => {
    render(<FileList files={[]} />);
    expect(screen.queryByTestId("file-list")).not.toBeInTheDocument();
  });

  it("renders file list container", () => {
    render(<FileList files={mockFiles} />);
    expect(screen.getByTestId("file-list")).toBeInTheDocument();
  });

  it("renders all file items", () => {
    render(<FileList files={mockFiles} />);
    expect(screen.getByTestId("file-item-f1")).toBeInTheDocument();
    expect(screen.getByTestId("file-item-f2")).toBeInTheDocument();
    expect(screen.getByTestId("file-item-f3")).toBeInTheDocument();
  });

  it("renders filenames", () => {
    render(<FileList files={mockFiles} />);
    expect(screen.getByText("report.pdf")).toBeInTheDocument();
    expect(screen.getByText("photo.png")).toBeInTheDocument();
  });

  it("renders file sizes", () => {
    render(<FileList files={mockFiles} />);
    expect(screen.getByText(/1\.0 MB/)).toBeInTheDocument();
    expect(screen.getByText(/2\.0 KB/)).toBeInTheDocument();
    expect(screen.getByText(/512 B/)).toBeInTheDocument();
  });

  it("renders image icon for image files", () => {
    render(<FileList files={mockFiles} />);
    expect(screen.getByTestId("image-icon-f2")).toBeInTheDocument();
  });

  it("renders file icon for non-image files", () => {
    render(<FileList files={mockFiles} />);
    expect(screen.getByTestId("file-icon-f1")).toBeInTheDocument();
    expect(screen.getByTestId("file-icon-f3")).toBeInTheDocument();
  });

  it("renders download links", () => {
    render(<FileList files={mockFiles} />);
    const link = screen.getByTestId("download-f1");
    expect(link.tagName).toBe("A");
    expect(link).toHaveAttribute("href", "/api/uploads/f1");
    expect(link).toHaveAttribute("download", "report.pdf");
  });

  it("renders download links with correct aria-labels", () => {
    render(<FileList files={mockFiles} />);
    expect(screen.getByLabelText("Download report.pdf")).toBeInTheDocument();
  });

  it("renders delete buttons when onDelete provided", () => {
    render(<FileList files={mockFiles} onDelete={vi.fn()} />);
    expect(screen.getByTestId("delete-f1")).toBeInTheDocument();
    expect(screen.getByTestId("delete-f2")).toBeInTheDocument();
  });

  it("hides delete buttons when onDelete not provided", () => {
    render(<FileList files={mockFiles} />);
    expect(screen.queryByTestId("delete-f1")).not.toBeInTheDocument();
  });

  it("calls onDelete with file ID when delete clicked", async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn();
    render(<FileList files={mockFiles} onDelete={onDelete} />);

    await user.click(screen.getByTestId("delete-f1"));
    expect(onDelete).toHaveBeenCalledWith("f1");
  });

  it("renders content types", () => {
    render(<FileList files={mockFiles} />);
    expect(screen.getByText(/application\/pdf/)).toBeInTheDocument();
    expect(screen.getByText(/image\/png/)).toBeInTheDocument();
  });
});
