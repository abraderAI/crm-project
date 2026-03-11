"use client";

import { useState } from "react";
import { Sidebar, type NavItem } from "./sidebar";
import { Topbar } from "./topbar";
import { Breadcrumbs, type BreadcrumbItem } from "./breadcrumbs";

interface AppLayoutProps {
  children: React.ReactNode;
  navItems: NavItem[];
  breadcrumbs?: BreadcrumbItem[];
  currentPath?: string;
  unreadCount?: number;
  userMenu?: React.ReactNode;
  onSearch?: (query: string) => void;
}

/** Main application shell layout with sidebar, topbar, breadcrumbs, and content area. */
export function AppLayout({
  children,
  navItems,
  breadcrumbs = [],
  currentPath,
  unreadCount = 0,
  userMenu,
  onSearch,
}: AppLayoutProps): React.ReactNode {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  return (
    <div
      className="flex h-screen overflow-hidden bg-background text-foreground"
      data-testid="app-layout"
    >
      <Sidebar
        items={navItems}
        currentPath={currentPath}
        collapsed={sidebarCollapsed}
        onToggle={() => setSidebarCollapsed(!sidebarCollapsed)}
      />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Topbar unreadCount={unreadCount} userMenu={userMenu} onSearch={onSearch} />
        <div className="flex-1 overflow-y-auto p-4">
          {breadcrumbs.length > 0 && (
            <div className="mb-4">
              <Breadcrumbs items={breadcrumbs} />
            </div>
          )}
          <main>{children}</main>
        </div>
      </div>
    </div>
  );
}
