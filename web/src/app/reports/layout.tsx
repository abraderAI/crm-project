"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { BarChart3, Headset } from "lucide-react";
import { cn } from "@/lib/utils";

const REPORT_TABS = [
  { href: "/reports/support", label: "Support", icon: Headset },
  { href: "/reports/sales", label: "Sales", icon: BarChart3 },
] as const;

export default function ReportsLayout({
  children,
}: {
  children: React.ReactNode;
}): React.ReactNode {
  const pathname = usePathname();

  return (
    <div data-testid="reports-layout" className="mx-auto max-w-7xl px-6 py-6">
      <h1 className="text-2xl font-bold text-foreground">Reports</h1>

      <div
        className="mt-4 flex gap-1 border-b border-border"
        data-testid="reports-tabs"
        role="tablist"
      >
        {REPORT_TABS.map(({ href, label, icon: Icon }) => {
          const isActive = pathname.startsWith(href);
          return (
            <Link
              key={href}
              href={href}
              role="tab"
              aria-selected={isActive}
              data-testid={`reports-tab-${label.toLowerCase()}`}
              className={cn(
                "inline-flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "border-primary text-foreground"
                  : "border-transparent text-muted-foreground hover:border-border hover:text-foreground",
              )}
            >
              <Icon className="h-4 w-4" />
              {label}
            </Link>
          );
        })}
      </div>

      <div className="mt-6">{children}</div>
    </div>
  );
}
