"use client";

import { useState } from "react";
import { ChatWidgetPreview } from "./chat-widget-preview";
import { EmbedCodeSnippet } from "./embed-code-snippet";

/** Props for the ChatChannelPanel component. */
export interface ChatChannelPanelProps {
  /** The org embed key for the embed code snippet. */
  embedKey: string;
}

/** Side panel for the chat channel config page showing live widget preview and embed code. */
export function ChatChannelPanel({ embedKey }: ChatChannelPanelProps): React.ReactNode {
  const [theme, setTheme] = useState("#3b82f6");
  const [greeting, setGreeting] = useState("Hello! How can we help you today?");
  const [logoUrl, setLogoUrl] = useState("");

  return (
    <div data-testid="chat-channel-panel" className="flex flex-col gap-6">
      {/* Customization fields */}
      <div className="flex flex-col gap-4 rounded-lg border border-border p-4">
        <h3 className="text-sm font-semibold text-foreground">Widget Appearance</h3>

        <div className="flex flex-col gap-1">
          <label htmlFor="widget-theme" className="text-xs font-medium text-foreground">
            Theme Color
          </label>
          <div className="flex items-center gap-2">
            <input
              id="widget-theme"
              data-testid="widget-theme-input"
              type="color"
              value={theme}
              onChange={(e) => setTheme(e.target.value)}
              className="h-8 w-8 cursor-pointer rounded border border-border"
            />
            <input
              data-testid="widget-theme-text"
              type="text"
              value={theme}
              onChange={(e) => setTheme(e.target.value)}
              className="rounded-md border border-border bg-background px-2 py-1 text-xs text-foreground"
            />
          </div>
        </div>

        <div className="flex flex-col gap-1">
          <label htmlFor="widget-greeting" className="text-xs font-medium text-foreground">
            Greeting Message
          </label>
          <input
            id="widget-greeting"
            data-testid="widget-greeting-input"
            type="text"
            value={greeting}
            onChange={(e) => setGreeting(e.target.value)}
            className="rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground"
          />
        </div>

        <div className="flex flex-col gap-1">
          <label htmlFor="widget-logo" className="text-xs font-medium text-foreground">
            Logo URL
          </label>
          <input
            id="widget-logo"
            data-testid="widget-logo-input"
            type="text"
            value={logoUrl}
            onChange={(e) => setLogoUrl(e.target.value)}
            placeholder="https://example.com/logo.png"
            className="rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground"
          />
        </div>
      </div>

      {/* Live preview */}
      <ChatWidgetPreview theme={theme} greeting={greeting} logoUrl={logoUrl} />

      {/* Embed code */}
      <EmbedCodeSnippet embedKey={embedKey} />
    </div>
  );
}
