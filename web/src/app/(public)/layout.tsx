import type { ReactNode } from "react";
import Link from "next/link";

/**
 * Public layout for unauthenticated routes (/docs, /forum).
 * No Clerk SignedIn wrapper — accessible to anonymous visitors.
 * Chatbot widget will be rendered here once Phase 6 is implemented.
 */
export default function PublicLayout({ children }: { children: ReactNode }): ReactNode {
  return (
    <div data-testid="public-layout" className="min-h-screen bg-background text-foreground">
      <header className="border-b border-border bg-background px-6 py-3">
        <Link href="/" className="text-lg font-bold text-foreground">
          DEFT Evolution
        </Link>
        <nav className="ml-6 inline-flex gap-4">
          <Link href="/docs" className="text-sm text-muted-foreground hover:text-foreground">
            Documentation
          </Link>
          <Link href="/forum" className="text-sm text-muted-foreground hover:text-foreground">
            Forum
          </Link>
          <Link href="/sign-in" className="text-sm text-muted-foreground hover:text-foreground">
            Sign In
          </Link>
        </nav>
      </header>
      <main className="mx-auto max-w-5xl p-6">{children}</main>
      {/* ChatbotWidget will be added in Phase 6 */}
    </div>
  );
}
