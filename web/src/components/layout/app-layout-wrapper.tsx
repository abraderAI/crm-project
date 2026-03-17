"use client";

import { useCallback, useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { useAuth, UserButton } from "@clerk/nextjs";
import dynamic from "next/dynamic";
import { Settings } from "lucide-react";

import { useNotifications } from "@/hooks/use-notifications";
import { TierProvider } from "@/hooks/use-tier";
import { AppLayout } from "./app-layout";
import type { NavItem } from "./sidebar";

/** Dynamically load ChatbotWidget without SSR to avoid hydration mismatches. */
const ChatbotWidget = dynamic(
  () => import("@/components/chatbot-widget").then((mod) => mod.ChatbotWidget),
  { ssr: false },
);

/** Small settings icon for the UserButton menu. */
function SettingsIcon(): React.ReactNode {
  return <Settings className="h-4 w-4" />;
}

/** Top-level navigation items rendered in the sidebar. */
const NAV_ITEMS: NavItem[] = [
  { id: "home", label: "Home", href: "/", type: "org" },
  { id: "forum", label: "Forum", href: "/forum", type: "org" },
  { id: "docs", label: "Docs", href: "/docs", type: "org" },
  { id: "support", label: "Support", href: "/support", type: "org" },
  { id: "notifications", label: "Notifications", href: "/notifications", type: "org" },
  { id: "crm", label: "CRM Pipeline", href: "/crm", type: "org" },
  { id: "reports", label: "Reports", href: "/reports", type: "org" },
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
    <TierProvider token={token}>
      <AppLayout
        navItems={NAV_ITEMS}
        currentPath={pathname}
        unreadCount={unreadCount}
        userMenu={
          <UserButton>
            <UserButton.MenuItems>
              <UserButton.Link label="Settings" labelIcon={<SettingsIcon />} href="/settings" />
            </UserButton.MenuItems>
          </UserButton>
        }
        onSearch={handleSearch}
      >
        {children}
        <ChatbotWidget />
      </AppLayout>
    </TierProvider>
  );
}
