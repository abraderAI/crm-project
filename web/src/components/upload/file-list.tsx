"use client";

import { Download, FileIcon, ImageIcon, Trash2 } from "lucide-react";

export interface FileItem {
  id: string;
  filename: string;
  contentType: string;
  size: number;
  downloadUrl: string;
}

export interface FileListProps {
  /** List of uploaded files. */
  files: FileItem[];
  /** Called when the user clicks delete on a file. */
  onDelete?: (fileId: string) => void;
}

/** Format file size to human-readable string. */
function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

/** Check if a content type is an image. */
function isImageType(contentType: string): boolean {
  return contentType.startsWith("image/");
}

/** List of uploaded files with download links and delete actions. */
export function FileList({ files, onDelete }: FileListProps): React.ReactNode {
  if (files.length === 0) {
    return (
      <p className="py-4 text-center text-sm text-muted-foreground" data-testid="no-files">
        No files uploaded.
      </p>
    );
  }

  return (
    <div data-testid="file-list" className="flex flex-col gap-2">
      {files.map((file) => (
        <div
          key={file.id}
          data-testid={`file-item-${file.id}`}
          className="flex items-center gap-3 rounded-lg border border-border p-3"
        >
          {/* Icon */}
          {isImageType(file.contentType) ? (
            <ImageIcon
              className="h-5 w-5 shrink-0 text-muted-foreground"
              data-testid={`image-icon-${file.id}`}
            />
          ) : (
            <FileIcon
              className="h-5 w-5 shrink-0 text-muted-foreground"
              data-testid={`file-icon-${file.id}`}
            />
          )}

          {/* File details */}
          <div className="flex-1 min-w-0">
            <p className="truncate text-sm font-medium text-foreground">{file.filename}</p>
            <p className="text-xs text-muted-foreground">
              {file.contentType} · {formatSize(file.size)}
            </p>
          </div>

          {/* Download */}
          <a
            href={file.downloadUrl}
            download={file.filename}
            data-testid={`download-${file.id}`}
            className="shrink-0 rounded p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
            aria-label={`Download ${file.filename}`}
          >
            <Download className="h-4 w-4" />
          </a>

          {/* Delete */}
          {onDelete && (
            <button
              onClick={() => onDelete(file.id)}
              data-testid={`delete-${file.id}`}
              className="shrink-0 rounded p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
              aria-label={`Delete ${file.filename}`}
            >
              <Trash2 className="h-4 w-4" />
            </button>
          )}
        </div>
      ))}
    </div>
  );
}
