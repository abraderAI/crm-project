"use client";

import { Download, FileText, Trash2 } from "lucide-react";
import type { Upload } from "@/lib/api-types";

export interface FilePreviewProps {
  upload: Upload;
  /** Full download URL for the file. */
  downloadUrl: string;
  /** Called when delete is clicked. */
  onDelete?: (uploadId: string) => void;
}

/** Check if a content type is an image. */
export function isImageType(contentType: string): boolean {
  return contentType.startsWith("image/");
}

/** Format file size to human-readable string. */
export function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

/** File preview with image thumbnail for images, download link, and optional delete. */
export function FilePreview({ upload, downloadUrl, onDelete }: FilePreviewProps): React.ReactNode {
  const isImage = isImageType(upload.content_type);

  return (
    <div
      className="flex items-center gap-3 rounded-md border border-border bg-background p-2"
      data-testid={`file-preview-${upload.id}`}
    >
      {/* Thumbnail or icon */}
      {isImage ? (
        <img
          src={downloadUrl}
          alt={upload.filename}
          className="h-12 w-12 rounded object-cover"
          data-testid={`file-thumb-${upload.id}`}
        />
      ) : (
        <div
          className="flex h-12 w-12 items-center justify-center rounded bg-muted"
          data-testid={`file-icon-${upload.id}`}
        >
          <FileText className="h-6 w-6 text-muted-foreground" />
        </div>
      )}

      {/* Info */}
      <div className="min-w-0 flex-1">
        <p
          className="truncate text-sm font-medium text-foreground"
          data-testid={`file-name-${upload.id}`}
        >
          {upload.filename}
        </p>
        <p className="text-xs text-muted-foreground" data-testid={`file-size-${upload.id}`}>
          {formatFileSize(upload.size)}
        </p>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-1">
        <a
          href={downloadUrl}
          download={upload.filename}
          aria-label={`Download ${upload.filename}`}
          data-testid={`file-download-${upload.id}`}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
        >
          <Download className="h-4 w-4" />
        </a>
        {onDelete && (
          <button
            type="button"
            onClick={() => onDelete(upload.id)}
            aria-label={`Delete ${upload.filename}`}
            data-testid={`file-delete-${upload.id}`}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        )}
      </div>
    </div>
  );
}
