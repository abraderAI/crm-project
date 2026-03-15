"use client";

import { useState } from "react";
import { Download, Loader2 } from "lucide-react";

export interface ExportButtonProps {
  /** Full export URL including query params. */
  url: string;
  /** Suggested filename for the download (e.g. "support-report.csv"). */
  filename: string;
}

/** Button that fetches a CSV export and triggers a browser download. */
export function ExportButton({ url, filename }: ExportButtonProps): React.ReactNode {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleClick(): Promise<void> {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(`Export failed: ${response.status}`);
      }

      const blob = await response.blob();
      const objectUrl = URL.createObjectURL(blob);

      const anchor = document.createElement("a");
      anchor.href = objectUrl;
      anchor.download = filename;
      document.body.appendChild(anchor);
      anchor.click();

      // Cleanup.
      document.body.removeChild(anchor);
      URL.revokeObjectURL(objectUrl);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Export failed";
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div data-testid="export-button-wrapper">
      <button
        type="button"
        onClick={() => void handleClick()}
        disabled={loading}
        data-testid="export-button"
        className="inline-flex items-center gap-2 rounded-md border border-border bg-background px-3 py-2 text-sm font-medium text-foreground transition-colors hover:bg-accent/50 disabled:opacity-50"
      >
        {loading ? (
          <Loader2 className="h-4 w-4 animate-spin" data-testid="export-spinner" />
        ) : (
          <Download className="h-4 w-4" data-testid="export-icon" />
        )}
        Export CSV
      </button>
      {error && (
        <p className="mt-1 text-xs text-red-600" data-testid="export-error">
          {error}
        </p>
      )}
    </div>
  );
}
