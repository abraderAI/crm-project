"use client";

import { Upload } from "lucide-react";
import { useCallback, useRef, useState } from "react";
import { cn } from "@/lib/utils";

/** Maximum file size in bytes (100MB). */
export const MAX_FILE_SIZE = 100 * 1024 * 1024;

/** Allowed file types (empty means all allowed). */
export const DEFAULT_ALLOWED_TYPES = [
  "image/jpeg",
  "image/png",
  "image/gif",
  "image/webp",
  "application/pdf",
  "text/plain",
  "text/csv",
  "application/json",
  "application/zip",
];

export interface FileValidationError {
  file: File;
  reason: string;
}

/** Validate a file against size and type constraints. */
export function validateFile(
  file: File,
  maxSize: number = MAX_FILE_SIZE,
  allowedTypes: string[] = DEFAULT_ALLOWED_TYPES,
): FileValidationError | null {
  if (file.size > maxSize) {
    return { file, reason: `File exceeds maximum size of ${Math.round(maxSize / 1024 / 1024)}MB` };
  }
  if (allowedTypes.length > 0 && !allowedTypes.includes(file.type)) {
    return { file, reason: `File type "${file.type || "unknown"}" is not allowed` };
  }
  return null;
}

export interface FileUploadProps {
  /** Called with valid files after selection/drop. */
  onFiles: (files: File[]) => void;
  /** Called with validation errors. */
  onError?: (errors: FileValidationError[]) => void;
  /** Maximum file size in bytes. */
  maxSize?: number;
  /** Allowed MIME types. */
  allowedTypes?: string[];
  /** Whether upload is disabled. */
  disabled?: boolean;
  /** Whether to accept multiple files. */
  multiple?: boolean;
}

/** Drag-and-drop file upload zone with client-side type/size validation. */
export function FileUpload({
  onFiles,
  onError,
  maxSize = MAX_FILE_SIZE,
  allowedTypes = DEFAULT_ALLOWED_TYPES,
  disabled = false,
  multiple = true,
}: FileUploadProps): React.ReactNode {
  const [isDragging, setIsDragging] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const processFiles = useCallback(
    (fileList: FileList | null) => {
      if (!fileList || fileList.length === 0) return;
      const files = Array.from(fileList);
      const valid: File[] = [];
      const errors: FileValidationError[] = [];
      for (const file of files) {
        const err = validateFile(file, maxSize, allowedTypes);
        if (err) {
          errors.push(err);
        } else {
          valid.push(file);
        }
      }
      if (valid.length > 0) onFiles(valid);
      if (errors.length > 0) onError?.(errors);
    },
    [onFiles, onError, maxSize, allowedTypes],
  );

  const handleDragOver = (e: React.DragEvent): void => {
    e.preventDefault();
    if (!disabled) setIsDragging(true);
  };

  const handleDragLeave = (): void => {
    setIsDragging(false);
  };

  const handleDrop = (e: React.DragEvent): void => {
    e.preventDefault();
    setIsDragging(false);
    if (!disabled) processFiles(e.dataTransfer.files);
  };

  const handleClick = (): void => {
    if (!disabled) inputRef.current?.click();
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>): void => {
    processFiles(e.target.files);
    if (inputRef.current) inputRef.current.value = "";
  };

  return (
    <div
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
      onClick={handleClick}
      data-testid="file-upload"
      className={cn(
        "flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed p-6 transition-colors",
        isDragging ? "border-primary bg-primary/5" : "border-border hover:border-primary/50",
        disabled && "cursor-not-allowed opacity-50",
      )}
    >
      <Upload className="mb-2 h-8 w-8 text-muted-foreground" />
      <p className="text-sm text-foreground">
        {isDragging ? "Drop files here" : "Drag & drop files, or click to browse"}
      </p>
      <p className="mt-1 text-xs text-muted-foreground">
        Max {Math.round(maxSize / 1024 / 1024)}MB per file
      </p>
      <input
        ref={inputRef}
        type="file"
        multiple={multiple}
        onChange={handleInputChange}
        className="hidden"
        data-testid="file-upload-input"
        disabled={disabled}
      />
    </div>
  );
}
