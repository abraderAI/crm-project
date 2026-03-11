"use client";

import { useCallback, useRef, useState } from "react";
import { Upload, X, FileIcon, ImageIcon } from "lucide-react";
import { cn } from "@/lib/utils";

/** Max file size: 100 MB. */
const MAX_FILE_SIZE = 100 * 1024 * 1024;

/** Allowed MIME type prefixes. */
const ALLOWED_TYPES = [
  "image/",
  "application/pdf",
  "text/",
  "application/json",
  "application/zip",
  "application/gzip",
  "application/msword",
  "application/vnd.openxmlformats",
];

export interface UploadedFile {
  file: File;
  preview?: string;
  progress: number;
  error?: string;
}

export interface FileUploadProps {
  /** Called when valid files are selected. */
  onUpload: (files: File[]) => void;
  /** Whether multiple files can be selected. */
  multiple?: boolean;
  /** Whether the upload zone is disabled. */
  disabled?: boolean;
  /** Max file size in bytes (default 100MB). */
  maxSize?: number;
  /** Accepted MIME type prefixes. */
  acceptedTypes?: string[];
}

/** Validate a file against size and type constraints. */
export function validateFile(file: File, maxSize: number, acceptedTypes: string[]): string | null {
  if (file.size > maxSize) {
    const sizeMB = Math.round(maxSize / (1024 * 1024));
    return `File exceeds ${sizeMB}MB limit`;
  }
  const isAllowed = acceptedTypes.some((type) => file.type.startsWith(type));
  if (!isAllowed && file.type !== "") {
    return `File type "${file.type}" is not allowed`;
  }
  return null;
}

/** Check if a file is an image. */
function isImage(file: File): boolean {
  return file.type.startsWith("image/");
}

/** File upload zone with drag-drop, validation, and image preview. */
export function FileUpload({
  onUpload,
  multiple = false,
  disabled = false,
  maxSize = MAX_FILE_SIZE,
  acceptedTypes = ALLOWED_TYPES,
}: FileUploadProps): React.ReactNode {
  const [isDragOver, setIsDragOver] = useState(false);
  const [stagedFiles, setStagedFiles] = useState<UploadedFile[]>([]);
  const inputRef = useRef<HTMLInputElement>(null);

  const processFiles = useCallback(
    (fileList: FileList | File[]): void => {
      const files = Array.from(fileList);
      const processed: UploadedFile[] = files.map((file) => {
        const error = validateFile(file, maxSize, acceptedTypes) ?? undefined;
        const preview = isImage(file) ? URL.createObjectURL(file) : undefined;
        return { file, preview, progress: 0, error };
      });
      setStagedFiles((prev) => (multiple ? [...prev, ...processed] : processed));

      const validFiles = processed.filter((f) => !f.error).map((f) => f.file);
      if (validFiles.length > 0) {
        onUpload(validFiles);
      }
    },
    [maxSize, acceptedTypes, multiple, onUpload],
  );

  const handleDrop = useCallback(
    (e: React.DragEvent): void => {
      e.preventDefault();
      setIsDragOver(false);
      if (disabled) return;
      processFiles(e.dataTransfer.files);
    },
    [disabled, processFiles],
  );

  const handleDragOver = (e: React.DragEvent): void => {
    e.preventDefault();
    if (!disabled) setIsDragOver(true);
  };

  const handleDragLeave = (): void => {
    setIsDragOver(false);
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>): void => {
    if (e.target.files) {
      processFiles(e.target.files);
    }
  };

  const removeFile = (index: number): void => {
    setStagedFiles((prev) => {
      const removed = prev[index];
      if (removed?.preview) URL.revokeObjectURL(removed.preview);
      return prev.filter((_, i) => i !== index);
    });
  };

  return (
    <div data-testid="file-upload" className="flex flex-col gap-3">
      {/* Drop zone */}
      <div
        data-testid="drop-zone"
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onClick={() => !disabled && inputRef.current?.click()}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            inputRef.current?.click();
          }
        }}
        className={cn(
          "flex cursor-pointer flex-col items-center gap-2 rounded-lg border-2 border-dashed p-6 transition-colors",
          isDragOver ? "border-primary bg-primary/5" : "border-border",
          disabled && "cursor-not-allowed opacity-50",
        )}
      >
        <Upload className="h-8 w-8 text-muted-foreground" />
        <p className="text-sm text-muted-foreground">
          Drag and drop files here, or click to browse
        </p>
        <p className="text-xs text-muted-foreground">Max {Math.round(maxSize / (1024 * 1024))}MB</p>
        <input
          ref={inputRef}
          type="file"
          multiple={multiple}
          onChange={handleInputChange}
          disabled={disabled}
          data-testid="file-input"
          className="hidden"
        />
      </div>

      {/* Staged files */}
      {stagedFiles.length > 0 && (
        <div className="flex flex-col gap-2" data-testid="staged-files">
          {stagedFiles.map((sf, index) => (
            <div
              key={`${sf.file.name}-${index}`}
              data-testid={`staged-file-${index}`}
              className={cn(
                "flex items-center gap-3 rounded-lg border p-3",
                sf.error ? "border-destructive/50 bg-destructive/5" : "border-border",
              )}
            >
              {/* Preview or icon */}
              {sf.preview ? (
                <img
                  src={sf.preview}
                  alt={sf.file.name}
                  data-testid={`file-preview-${index}`}
                  className="h-10 w-10 shrink-0 rounded object-cover"
                />
              ) : (
                <FileIcon className="h-8 w-8 shrink-0 text-muted-foreground" />
              )}

              <div className="flex-1 min-w-0">
                <p className="truncate text-sm font-medium text-foreground">{sf.file.name}</p>
                <p className="text-xs text-muted-foreground">
                  {(sf.file.size / 1024).toFixed(1)} KB
                </p>
                {sf.error && (
                  <p className="text-xs text-destructive" data-testid={`file-error-${index}`}>
                    {sf.error}
                  </p>
                )}
              </div>

              {/* Remove */}
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  removeFile(index);
                }}
                aria-label={`Remove ${sf.file.name}`}
                data-testid={`remove-file-${index}`}
                className="shrink-0 rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export { isImage, MAX_FILE_SIZE, ALLOWED_TYPES, type ImageIcon };
