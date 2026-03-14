import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { FileUpload, validateFile, ALLOWED_TYPES, MAX_FILE_SIZE } from "./file-upload";

// Mock URL.createObjectURL/revokeObjectURL.
beforeEach(() => {
  vi.stubGlobal("URL", {
    ...globalThis.URL,
    createObjectURL: vi.fn().mockReturnValue("blob:mock-url"),
    revokeObjectURL: vi.fn(),
  });
});

function createMockFile(name: string, size: number, type: string): File {
  const buffer = new ArrayBuffer(size);
  return new File([buffer], name, { type });
}

describe("validateFile", () => {
  it("returns null for valid file", () => {
    const file = createMockFile("test.txt", 1024, "text/plain");
    expect(validateFile(file, MAX_FILE_SIZE, ALLOWED_TYPES)).toBeNull();
  });

  it("returns error for oversized file", () => {
    const file = createMockFile("big.txt", MAX_FILE_SIZE + 1, "text/plain");
    expect(validateFile(file, MAX_FILE_SIZE, ALLOWED_TYPES)).toContain("exceeds");
  });

  it("returns error for disallowed type", () => {
    const file = createMockFile("bad.exe", 1024, "application/x-executable");
    expect(validateFile(file, MAX_FILE_SIZE, ALLOWED_TYPES)).toContain("not allowed");
  });

  it("allows files with empty type", () => {
    const file = createMockFile("unknown", 1024, "");
    expect(validateFile(file, MAX_FILE_SIZE, ALLOWED_TYPES)).toBeNull();
  });

  it("allows image files", () => {
    const file = createMockFile("photo.png", 1024, "image/png");
    expect(validateFile(file, MAX_FILE_SIZE, ALLOWED_TYPES)).toBeNull();
  });

  it("allows PDF files", () => {
    const file = createMockFile("doc.pdf", 1024, "application/pdf");
    expect(validateFile(file, MAX_FILE_SIZE, ALLOWED_TYPES)).toBeNull();
  });
});

describe("FileUpload", () => {
  it("renders upload container", () => {
    render(<FileUpload onUpload={vi.fn()} />);
    expect(screen.getByTestId("file-upload")).toBeInTheDocument();
  });

  it("renders drop zone", () => {
    render(<FileUpload onUpload={vi.fn()} />);
    expect(screen.getByTestId("drop-zone")).toBeInTheDocument();
  });

  it("renders instructions text", () => {
    render(<FileUpload onUpload={vi.fn()} />);
    expect(screen.getByText(/Drag and drop/)).toBeInTheDocument();
  });

  it("shows max file size", () => {
    render(<FileUpload onUpload={vi.fn()} />);
    expect(screen.getByText(/Max 100MB/)).toBeInTheDocument();
  });

  it("shows custom max size", () => {
    render(<FileUpload onUpload={vi.fn()} maxSize={10 * 1024 * 1024} />);
    expect(screen.getByText(/Max 10MB/)).toBeInTheDocument();
  });

  it("calls onUpload when file input changes", async () => {
    const onUpload = vi.fn();
    render(<FileUpload onUpload={onUpload} />);

    const file = createMockFile("test.txt", 1024, "text/plain");
    const input = screen.getByTestId("file-input");
    await userEvent.upload(input, file);

    expect(onUpload).toHaveBeenCalledWith([file]);
  });

  it("shows staged file after upload", async () => {
    render(<FileUpload onUpload={vi.fn()} />);

    const file = createMockFile("test.txt", 1024, "text/plain");
    const input = screen.getByTestId("file-input");
    await userEvent.upload(input, file);

    expect(screen.getByTestId("staged-files")).toBeInTheDocument();
    expect(screen.getByTestId("staged-file-0")).toBeInTheDocument();
    expect(screen.getByText("test.txt")).toBeInTheDocument();
  });

  it("renders FilePreview for staged file", async () => {
    render(<FileUpload onUpload={vi.fn()} />);

    const file = createMockFile("report.pdf", 2048, "application/pdf");
    await userEvent.upload(screen.getByTestId("file-input"), file);

    // FilePreview renders with temp upload id "staged-0"
    expect(screen.getByTestId("file-preview-staged-0")).toBeInTheDocument();
    expect(screen.getByTestId("file-name-staged-0")).toHaveTextContent("report.pdf");
    expect(screen.getByTestId("file-size-staged-0")).toHaveTextContent("2.0 KB");
  });

  it("renders FilePreview download link for staged file", async () => {
    render(<FileUpload onUpload={vi.fn()} />);

    const file = createMockFile("report.pdf", 1024, "application/pdf");
    await userEvent.upload(screen.getByTestId("file-input"), file);

    const downloadLink = screen.getByTestId("file-download-staged-0");
    expect(downloadLink).toHaveAttribute("download", "report.pdf");
  });

  it("shows file size via FilePreview", async () => {
    render(<FileUpload onUpload={vi.fn()} />);

    const file = createMockFile("test.txt", 2048, "text/plain");
    await userEvent.upload(screen.getByTestId("file-input"), file);

    expect(screen.getByTestId("file-size-staged-0")).toHaveTextContent("2.0 KB");
  });

  it("shows error for invalid file", async () => {
    render(<FileUpload onUpload={vi.fn()} maxSize={100} />);

    const file = createMockFile("big.txt", 200, "text/plain");
    await userEvent.upload(screen.getByTestId("file-input"), file);

    expect(screen.getByTestId("file-error-0")).toBeInTheDocument();
  });

  it("does not call onUpload for invalid files", async () => {
    const onUpload = vi.fn();
    render(<FileUpload onUpload={onUpload} maxSize={100} />);

    const file = createMockFile("big.txt", 200, "text/plain");
    await userEvent.upload(screen.getByTestId("file-input"), file);

    expect(onUpload).not.toHaveBeenCalled();
  });

  it("shows image thumbnail via FilePreview for image files", async () => {
    render(<FileUpload onUpload={vi.fn()} />);

    const file = createMockFile("photo.png", 1024, "image/png");
    await userEvent.upload(screen.getByTestId("file-input"), file);

    expect(screen.getByTestId("file-thumb-staged-0")).toBeInTheDocument();
    expect(screen.getByTestId("file-thumb-staged-0")).toHaveAttribute("src", "blob:mock-url");
  });

  it("shows file icon via FilePreview for non-image files", async () => {
    render(<FileUpload onUpload={vi.fn()} />);

    const file = createMockFile("report.pdf", 1024, "application/pdf");
    await userEvent.upload(screen.getByTestId("file-input"), file);

    expect(screen.getByTestId("file-icon-staged-0")).toBeInTheDocument();
  });

  it("removes file via FilePreview delete button", async () => {
    const user = userEvent.setup();
    render(<FileUpload onUpload={vi.fn()} />);

    const file = createMockFile("test.txt", 1024, "text/plain");
    await userEvent.upload(screen.getByTestId("file-input"), file);

    expect(screen.getByTestId("staged-file-0")).toBeInTheDocument();

    await user.click(screen.getByTestId("file-delete-staged-0"));
    expect(screen.queryByTestId("staged-file-0")).not.toBeInTheDocument();
  });

  it("handles drag and drop", () => {
    const onUpload = vi.fn();
    render(<FileUpload onUpload={onUpload} />);

    const dropZone = screen.getByTestId("drop-zone");
    const file = createMockFile("drop.txt", 1024, "text/plain");

    fireEvent.dragOver(dropZone);
    fireEvent.drop(dropZone, {
      dataTransfer: { files: [file] },
    });

    expect(onUpload).toHaveBeenCalledWith([file]);
  });

  it("handles drag leave", () => {
    render(<FileUpload onUpload={vi.fn()} />);
    const dropZone = screen.getByTestId("drop-zone");

    fireEvent.dragOver(dropZone);
    fireEvent.dragLeave(dropZone);

    // Should not crash.
    expect(dropZone).toBeInTheDocument();
  });

  it("does not process files when disabled", () => {
    const onUpload = vi.fn();
    render(<FileUpload onUpload={onUpload} disabled={true} />);

    const dropZone = screen.getByTestId("drop-zone");
    const file = createMockFile("test.txt", 1024, "text/plain");

    fireEvent.drop(dropZone, {
      dataTransfer: { files: [file] },
    });

    expect(onUpload).not.toHaveBeenCalled();
  });
});
