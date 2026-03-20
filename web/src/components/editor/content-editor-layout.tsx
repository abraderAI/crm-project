"use client";

import type { ReactNode } from "react";

/** Props for the ContentEditorLayout shell. */
export interface ContentEditorLayoutProps {
  /** Header area — typically title, ID badge, status control. */
  header: ReactNode;
  /** Main body area — timeline or editor content. */
  children: ReactNode;
  /** Optional sidebar — metadata, attachments, preferences. */
  sidebar?: ReactNode;
  /** Optional composer area rendered below the timeline. */
  composer?: ReactNode;
}

/**
 * ContentEditorLayout is the shared presentational shell for support tickets,
 * forum threads, wiki articles, and documentation pages.
 *
 * It provides a consistent two-column layout:
 *   - Left column: header, timeline/content body, composer
 *   - Right column: sidebar (metadata, attachments, preferences)
 *
 * The layout is fully responsive — the sidebar stacks below on small screens.
 */
export function ContentEditorLayout({
  header,
  children,
  sidebar,
  composer,
}: ContentEditorLayoutProps): ReactNode {
  return (
    <div data-testid="content-editor-layout" className="flex flex-col gap-4">
      {/* Header */}
      <div data-testid="editor-header" className="rounded-lg border border-border bg-background p-4">
        {header}
      </div>

      {/* Two-column body */}
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start">
        {/* Main content column */}
        <div className="flex min-w-0 flex-1 flex-col gap-4">
          <div data-testid="editor-body">{children}</div>
          {composer && (
            <div data-testid="editor-composer" className="rounded-lg border border-border bg-background">
              {composer}
            </div>
          )}
        </div>

        {/* Sidebar column */}
        {sidebar && (
          <aside
            data-testid="editor-sidebar"
            className="w-full shrink-0 lg:w-72"
          >
            {sidebar}
          </aside>
        )}
      </div>
    </div>
  );
}
