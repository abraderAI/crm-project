"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { useAuth, UserButton } from "@clerk/nextjs";
import dynamic from "next/dynamic";
import { Settings } from "lucide-react";

import { useNotifications } from "@/hooks/use-notifications";
import { TierProvider, useTier } from "@/hooks/use-tier";
import { getNavItemsForTier } from "@/lib/nav-config";
import { AppLayout } from "./app-layout";

/** Dynamically load ChatbotWidget without SSR to avoid hydration mismatches. */
const ChatbotWidget = dynamic(
  () => import("@/components/chatbot-widget").then((mod) => mod.ChatbotWidget),
  { ssr: false },
);

/** Small settings icon for the UserButton menu. */
function SettingsIcon(): React.ReactNode {
  return <Settings className="h-4 w-4" />;
}

interface AppLayoutWrapperProps {
  children: React.ReactNode;
}

/** Inner layout that consumes tier context to filter nav items. */
function AppLayoutInner({ children }: AppLayoutWrapperProps): React.ReactNode {
  const pathname = usePathname();
  const router = useRouter();
  const { getToken } = useAuth();
  const { tier } = useTier();

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

  /** Nav items filtered by the user's tier. */
  const navItems = useMemo(() => getNavItemsForTier(tier), [tier]);

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
      navItems={navItems}
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
  );
}

/** Client wrapper wiring AppLayout with Clerk auth, tier filtering, notifications, and routing. */
export function AppLayoutWrapper({ children }: AppLayoutWrapperProps): React.ReactNode {
  const { getToken } = useAuth();
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

  return (
    <TierProvider token={token}>
      <AppLayoutInner>{children}</AppLayoutInner>
    </TierProvider>
  );
}
