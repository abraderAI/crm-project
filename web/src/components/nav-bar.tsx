"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { UserButton } from "@clerk/nextjs";
import { LayoutDashboard, Shield, BarChart3, Bell, Search } from "lucide-react";
import { cn } from "@/lib/utils";

const NAV_ITEMS = [
  { href: "/", label: "Home", icon: LayoutDashboard },
  { href: "/crm", label: "CRM", icon: BarChart3 },
  { href: "/notifications", label: "Notifications", icon: Bell },
  { href: "/search", label: "Search", icon: Search },
  { href: "/admin", label: "Admin", icon: Shield },
] as const;

/** Top-level navigation bar with Clerk user button. */
export function NavBar(): React.ReactNode {
  const pathname = usePathname();

  return (
    <nav
      data-testid="nav-bar"
      className="flex items-center gap-6 border-b border-border bg-background px-6 py-3"
    >
      <Link href="/" className="text-lg font-bold text-foreground" data-testid="nav-logo">
        DEFT Evolution
      </Link>

      <div className="flex items-center gap-1">
        {NAV_ITEMS.map(({ href, label, icon: Icon }) => {
          const isActive = href === "/" ? pathname === "/" : pathname.startsWith(href);
          return (
            <Link
              key={href}
              href={href}
              data-testid={`nav-link-${label.toLowerCase()}`}
              className={cn(
                "inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
                isActive
                  ? "bg-accent text-foreground"
                  : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
              )}
            >
              <Icon className="h-4 w-4" />
              {label}
            </Link>
          );
        })}
      </div>

      <div className="ml-auto" data-testid="nav-user-button">
        <UserButton />
      </div>
    </nav>
  );
}
