"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Clipboard, Check } from "lucide-react";

/** Props for the EmbedCodeSnippet component. */
export interface EmbedCodeSnippetProps {
  /** The org-specific embed key injected into the script tag. */
  embedKey: string;
}

/** Generates the embed code snippet for the chat widget. */
function buildSnippet(embedKey: string): string {
  return `<script src="https://cdn.deft.dev/widget.js" data-org-key="${embedKey}"></script>`;
}

/** Displays a copyable embed code snippet for the chat widget. */
export function EmbedCodeSnippet({ embedKey }: EmbedCodeSnippetProps): React.ReactNode {
  const [copied, setCopied] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const snippet = buildSnippet(embedKey);

  useEffect(() => {
    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, []);

  const handleCopy = useCallback(async (): Promise<void> => {
    try {
      await navigator.clipboard.writeText(snippet);
      setCopied(true);
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
      timerRef.current = setTimeout(() => {
        setCopied(false);
        timerRef.current = null;
      }, 2000);
    } catch {
      // Clipboard API may fail in non-secure contexts; silently ignore.
    }
  }, [snippet]);

  return (
    <div data-testid="embed-code-snippet" className="flex flex-col gap-2">
      <h3 className="text-sm font-semibold text-foreground">Embed Code</h3>
      <p className="text-xs text-muted-foreground">
        Add this snippet to your website to enable the chat widget.
      </p>

      <div className="relative rounded-md border border-border bg-muted/50 p-3">
        <pre className="overflow-x-auto text-xs">
          <code data-testid="embed-code-text">{snippet}</code>
        </pre>

        <div className="mt-2 flex items-center gap-2">
          <button
            data-testid="embed-copy-btn"
            type="button"
            onClick={() => void handleCopy()}
            className="inline-flex items-center gap-1 rounded-md border border-border bg-background px-3 py-1.5 text-xs font-medium text-foreground hover:bg-accent"
          >
            {copied ? (
              <Check className="h-3 w-3 text-green-500" />
            ) : (
              <Clipboard className="h-3 w-3" />
            )}
            {copied ? "Copied" : "Copy"}
          </button>

          {copied && (
            <span data-testid="embed-copied-msg" className="text-xs font-medium text-green-600">
              Copied!
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
