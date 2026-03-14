"use client";

import { useCallback, useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { useAuth, UserButton } from "@clerk/nextjs";

import { useNotifications } from "@/hooks/use-notifications";
import { AppLayout } from "./app-layout";
import type { NavItem } from "./sidebar";

/** Top-level navigation items rendered in the sidebar. */
const NAV_ITEMS: NavItem[] = [
  { id: "home", label: "Home", href: "/", type: "org" },
  { id: "crm", label: "CRM Pipeline", href: "/crm", type: "org" },
  { id: "search", label: "Search", href: "/search", type: "org" },
  { id: "admin", label: "Admin", href: "/admin", type: "org" },
];

interface AppLayoutWrapperProps {
  children: React.ReactNode;
}

/** Client wrapper wiring AppLayout with Clerk auth, notifications, and routing. */
export function AppLayoutWrapper({ children }: AppLayoutWrapperProps): React.ReactNode {
  const pathname = usePathname();
  const router = useRouter();
  const { getToken } = useAuth();

  // Notification unread count.
  const [token, setToken] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    getToken().then((t) => {
      if (active) setToken(t);
    });
    return () => {
      active = false;
    };
  }, [getToken]);

  const { unreadCount } = useNotifications({ token });

  const handleSearch = useCallback(
    (query: string) => {
      if (!query.trim()) return;
      const params = new URLSearchParams({ q: query });
      router.push(`/search?${params.toString()}`);
    },
    [router],
  );

  return (
    <AppLayout
      navItems={NAV_ITEMS}
      currentPath={pathname}
      unreadCount={unreadCount}
      userMenu={<UserButton />}
      onSearch={handleSearch}
    >
      {children}
    </AppLayout>
  );
}
