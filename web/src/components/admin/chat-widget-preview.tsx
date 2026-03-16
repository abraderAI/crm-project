"use client";

import { useState } from "react";
import { MessageCircle, X } from "lucide-react";

/** Props for the ChatWidgetPreview component. */
export interface ChatWidgetPreviewProps {
  /** Primary theme color (hex). */
  theme: string;
  /** Greeting text shown in the chat panel header. */
  greeting: string;
  /** URL for the logo displayed in the chat panel. */
  logoUrl: string;
}

/** Live preview simulation of the embeddable chat widget. */
export function ChatWidgetPreview({
  theme,
  greeting,
  logoUrl,
}: ChatWidgetPreviewProps): React.ReactNode {
  const [expanded, setExpanded] = useState(false);

  return (
    <div data-testid="chat-widget-preview" className="flex flex-col gap-3">
      <h3 className="text-sm font-semibold text-foreground">Widget Preview</h3>

      <div className="relative h-[400px] w-full rounded-lg border border-border bg-muted/30">
        {/* Simulated page background */}
        <div className="flex h-full items-end justify-end p-4">
          {expanded && (
            <div
              data-testid="chat-widget-panel"
              className="absolute bottom-16 right-4 flex h-[320px] w-[280px] flex-col rounded-lg border border-border bg-background shadow-lg"
            >
              {/* Panel header */}
              <div
                data-testid="chat-widget-panel-header"
                className="flex items-center gap-2 rounded-t-lg px-3 py-2 text-white"
                style={{ backgroundColor: theme }}
              >
                {logoUrl && (
                  <img
                    data-testid="chat-widget-logo"
                    src={logoUrl}
                    alt="Chat logo"
                    className="h-6 w-6 rounded-full object-cover"
                  />
                )}
                <span className="flex-1 text-sm font-medium">Chat</span>
                <button
                  data-testid="chat-widget-close"
                  type="button"
                  onClick={() => setExpanded(false)}
                  className="rounded p-0.5 hover:bg-white/20"
                  aria-label="Close chat"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>

              {/* Greeting */}
              <div className="flex-1 overflow-y-auto p-3">
                <div
                  data-testid="chat-widget-greeting"
                  className="rounded-lg bg-muted px-3 py-2 text-sm text-foreground"
                >
                  {greeting}
                </div>
              </div>

              {/* Mock input */}
              <div className="border-t border-border p-2">
                <input
                  data-testid="chat-widget-input"
                  type="text"
                  placeholder="Type a message..."
                  disabled
                  className="w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-muted-foreground"
                />
              </div>
            </div>
          )}

          {/* Floating button */}
          <button
            data-testid="chat-widget-button"
            type="button"
            onClick={() => setExpanded(!expanded)}
            style={{ backgroundColor: theme }}
            className="flex h-12 w-12 items-center justify-center rounded-full text-white shadow-lg transition-transform hover:scale-105"
            aria-label="Open chat"
          >
            <MessageCircle className="h-6 w-6" />
          </button>
        </div>
      </div>
    </div>
  );
}
