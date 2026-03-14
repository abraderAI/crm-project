"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  BarChart3,
  Users,
  ScrollText,
  ToggleRight,
  CreditCard,
  Webhook,
  UserCog,
} from "lucide-react";
import { cn } from "@/lib/utils";

const ADMIN_TABS = [
  { href: "/admin", label: "Overview", icon: BarChart3, exact: true },
  { href: "/admin/users", label: "Users", icon: Users, exact: false },
  { href: "/admin/audit-log", label: "Audit Log", icon: ScrollText, exact: false },
  { href: "/admin/feature-flags", label: "Feature Flags", icon: ToggleRight, exact: false },
  { href: "/admin/billing", label: "Billing", icon: CreditCard, exact: false },
  { href: "/admin/webhooks", label: "Webhooks", icon: Webhook, exact: false },
  { href: "/admin/members", label: "Members", icon: UserCog, exact: false },
] as const;

export default function AdminLayout({ children }: { children: React.ReactNode }): React.ReactNode {
  const pathname = usePathname();

  return (
    <div data-testid="admin-layout" className="mx-auto max-w-7xl px-6 py-6">
      <h1 className="text-2xl font-bold text-foreground">Admin Dashboard</h1>

      <div
        className="mt-4 flex gap-1 border-b border-border"
        data-testid="admin-tabs"
        role="tablist"
      >
        {ADMIN_TABS.map(({ href, label, icon: Icon, exact }) => {
          const isActive = exact ? pathname === href : pathname.startsWith(href);
          return (
            <Link
              key={href}
              href={href}
              role="tab"
              aria-selected={isActive}
              data-testid={`admin-tab-${label.toLowerCase().replace(/\s+/g, "-")}`}
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
