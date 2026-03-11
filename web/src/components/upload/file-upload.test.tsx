import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { FileUpload, MAX_FILE_SIZE, validateFile } from "./file-upload";

function createFile(name: string, size: number, type: string): File {
  const buffer = new ArrayBuffer(size);
  return new File([buffer], name, { type });
}

describe("validateFile", () => {
  it("returns null for valid file", () => {
    const file = createFile("photo.jpg", 1024, "image/jpeg");
    expect(validateFile(file)).toBeNull();
  });

  it("returns error for oversized file", () => {
    const file = createFile("huge.jpg", MAX_FILE_SIZE + 1, "image/jpeg");
    const err = validateFile(file);
    expect(err).not.toBeNull();
    expect(err?.reason).toContain("exceeds maximum size");
  });

  it("returns error for disallowed type", () => {
    const file = createFile("script.exe", 100, "application/x-msdownload");
    const err = validateFile(file);
    expect(err).not.toBeNull();
    expect(err?.reason).toContain("not allowed");
  });

  it("returns null when allowed types is empty (all allowed)", () => {
    const file = createFile("anything.xyz", 100, "application/octet-stream");
    expect(validateFile(file, MAX_FILE_SIZE, [])).toBeNull();
  });

  it("handles file with no type", () => {
    const file = createFile("noext", 100, "");
    const err = validateFile(file);
    expect(err).not.toBeNull();
    expect(err?.reason).toContain("unknown");
  });

  it("uses custom max size", () => {
    const file = createFile("small.txt", 200, "text/plain");
    expect(validateFile(file, 100)).not.toBeNull();
    expect(validateFile(file, 300)).toBeNull();
  });
});

describe("FileUpload", () => {
  it("renders the upload zone", () => {
    render(<FileUpload onFiles={vi.fn()} />);
    expect(screen.getByTestId("file-upload")).toBeInTheDocument();
    expect(screen.getByText("Drag & drop files, or click to browse")).toBeInTheDocument();
  });

  it("shows max size hint", () => {
    render(<FileUpload onFiles={vi.fn()} />);
    expect(screen.getByText("Max 100MB per file")).toBeInTheDocument();
  });

  it("opens file dialog on click", () => {
    render(<FileUpload onFiles={vi.fn()} />);
    const input = screen.getByTestId("file-upload-input") as HTMLInputElement;
    const clickSpy = vi.spyOn(input, "click");
    fireEvent.click(screen.getByTestId("file-upload"));
    expect(clickSpy).toHaveBeenCalled();
  });

  it("calls onFiles with valid files from input", () => {
    const onFiles = vi.fn();
    render(<FileUpload onFiles={onFiles} />);

    const input = screen.getByTestId("file-upload-input");
    const file = createFile("photo.jpg", 1024, "image/jpeg");
    fireEvent.change(input, { target: { files: [file] } });
    expect(onFiles).toHaveBeenCalledWith([file]);
  });

  it("calls onError for invalid files from input", () => {
    const onFiles = vi.fn();
    const onError = vi.fn();
    render(<FileUpload onFiles={onFiles} onError={onError} />);

    const input = screen.getByTestId("file-upload-input");
    const file = createFile("script.exe", 100, "application/x-msdownload");
    fireEvent.change(input, { target: { files: [file] } });
    expect(onFiles).not.toHaveBeenCalled();
    expect(onError).toHaveBeenCalledWith(
      expect.arrayContaining([
        expect.objectContaining({ reason: expect.stringContaining("not allowed") }),
      ]),
    );
  });

  it("handles drag over event", () => {
    render(<FileUpload onFiles={vi.fn()} />);
    const zone = screen.getByTestId("file-upload");
    fireEvent.dragOver(zone, { dataTransfer: { files: [] } });
    expect(screen.getByText("Drop files here")).toBeInTheDocument();
  });

  it("handles drag leave event", () => {
    render(<FileUpload onFiles={vi.fn()} />);
    const zone = screen.getByTestId("file-upload");
    fireEvent.dragOver(zone, { dataTransfer: { files: [] } });
    fireEvent.dragLeave(zone);
    expect(screen.getByText("Drag & drop files, or click to browse")).toBeInTheDocument();
  });

  it("handles drop event with valid files", () => {
    const onFiles = vi.fn();
    render(<FileUpload onFiles={onFiles} />);
    const zone = screen.getByTestId("file-upload");
    const file = createFile("photo.png", 1024, "image/png");
    fireEvent.drop(zone, { dataTransfer: { files: [file] } });
    expect(onFiles).toHaveBeenCalledWith([file]);
  });

  it("does not process files when disabled", () => {
    const onFiles = vi.fn();
    render(<FileUpload onFiles={onFiles} disabled={true} />);
    const zone = screen.getByTestId("file-upload");
    const file = createFile("photo.png", 1024, "image/png");
    fireEvent.drop(zone, { dataTransfer: { files: [file] } });
    expect(onFiles).not.toHaveBeenCalled();
  });

  it("disables file input when disabled", () => {
    render(<FileUpload onFiles={vi.fn()} disabled={true} />);
    expect(screen.getByTestId("file-upload-input")).toBeDisabled();
  });
});
