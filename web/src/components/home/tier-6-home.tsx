"use client";

import type { ReactNode } from "react";
import Link from "next/link";
import type { WidgetConfig } from "@/lib/tier-types";

export interface Tier6HomeScreenProps {
  /** Auth token for API calls. */
  token: string;
  /** Ordered widget layout (from preferences or default). */
  layout: WidgetConfig[];
}

/** Admin quick-link definition. */
interface AdminLink {
  label: string;
  href: string;
  testId: string;
}

const ADMIN_LINKS: AdminLink[] = [
  { label: "Admin Dashboard", href: "/admin", testId: "link-admin-dashboard" },
  { label: "Support Report", href: "/admin/reports/support", testId: "link-support-report" },
  { label: "Sales Report", href: "/admin/reports/sales", testId: "link-sales-report" },
  { label: "User Management", href: "/admin/users", testId: "link-user-management" },
  { label: "Feature Flags", href: "/admin/feature-flags", testId: "link-feature-flags" },
  { label: "Audit Log", href: "/admin/audit-log", testId: "link-audit-log" },
];

/**
 * Tier 6 (Platform Admin) home screen.
 * Lightweight navigation hub pointing to the admin console where real data lives.
 */
export function Tier6HomeScreen({
  token: _token,
  layout: _layout,
}: Tier6HomeScreenProps): ReactNode {
  return (
    <div data-testid="tier6-home-screen" className="space-y-4">
      <div className="flex items-center gap-2">
        <h2 className="text-lg font-semibold text-foreground">Platform Admin Dashboard</h2>
        <span
          className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-800"
          data-testid="tier6-admin-badge"
        >
          Admin
        </span>
      </div>

      <p className="text-sm text-muted-foreground">
        Manage the platform from the{" "}
        <Link href="/admin" className="underline hover:text-foreground">
          Admin Console
        </Link>
        .
      </p>

      {/* Quick links */}
      <div className="flex flex-wrap gap-3" data-testid="tier6-quick-links">
        {ADMIN_LINKS.map((link) => (
          <Link
            key={link.href}
            href={link.href}
            className="rounded-md border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-accent/50"
            data-testid={link.testId}
          >
            {link.label}
          </Link>
        ))}
      </div>
    </div>
  );
}
