"use client";

import {
  ChevronDown,
  ChevronRight,
  LayoutDashboard,
  PanelLeftClose,
  PanelLeft,
} from "lucide-react";
import { useState } from "react";
import { cn } from "@/lib/utils";
import type { SidebarNavItem } from "@/lib/nav-config";

/**
 * @deprecated Use SidebarNavItem from nav-config.ts instead.
 * Kept for backward compatibility with tests and consumers.
 */
export type NavItem = SidebarNavItem;

export interface SidebarProps {
  items: SidebarNavItem[];
  currentPath?: string;
  collapsed?: boolean;
  onToggle?: () => void;
}

/** Determine which top-level section should be auto-expanded based on the current route. */
function findActiveParentId(items: SidebarNavItem[], currentPath?: string): string | null {
  if (!currentPath) return null;
  for (const item of items) {
    if (item.children?.some((child) => currentPath === child.href || currentPath.startsWith(child.href + "/"))) {
      return item.id;
    }
    if (currentPath === item.href || currentPath.startsWith(item.href + "/")) {
      if (item.children && item.children.length > 0) return item.id;
    }
  }
  return null;
}

/**
 * Animated sub-menu container.
 * Uses a large max-height when expanded so the CSS transition smoothly reveals content.
 */
function SubMenuPanel({
  expanded,
  children,
}: {
  expanded: boolean;
  children: React.ReactNode;
}): React.ReactNode {
  return (
    <ul
      role="group"
      className="sidebar-submenu overflow-hidden"
      style={{ maxHeight: expanded ? "2000px" : "0px" }}
      data-testid="submenu-panel"
      data-expanded={expanded}
    >
      {children}
    </ul>
  );
}

/** Single child item rendered inside a sub-menu. */
function SubMenuItem({
  item,
  currentPath,
}: {
  item: SidebarNavItem;
  currentPath?: string;
}): React.ReactNode {
  const isActive = currentPath === item.href;
  const Icon = item.icon;

  return (
    <li data-testid={`nav-item-${item.id}`}>
      <a
        href={item.href}
        data-testid={`nav-link-${item.id}`}
        className={cn(
          "flex items-center gap-2 rounded-md py-1.5 pl-10 pr-2 text-sm transition-colors duration-150",
          isActive
            ? "bg-primary/10 font-medium text-primary"
            : "text-foreground/60 hover:bg-foreground/5 hover:text-foreground",
        )}
      >
        <Icon className="h-3.5 w-3.5 shrink-0" />
        <span className="truncate">{item.label}</span>
      </a>
    </li>
  );
}

/** Top-level navigation item with optional expandable sub-menu. */
function TopLevelItem({
  item,
  currentPath,
  isExpanded,
  onToggleExpand,
}: {
  item: SidebarNavItem;
  currentPath?: string;
  isExpanded: boolean;
  onToggleExpand: () => void;
}): React.ReactNode {
  const hasChildren = item.children && item.children.length > 0;
  const isActive =
    currentPath === item.href ||
    (hasChildren && item.children!.some((c) => currentPath === c.href || (currentPath?.startsWith(c.href + "/") ?? false)));
  const Icon = item.icon;

  return (
    <li data-testid={`nav-item-${item.id}`}>
      <div
        className={cn(
          "group flex items-center gap-2 rounded-md px-2 py-2 text-sm transition-colors duration-150",
          isActive
            ? "sidebar-active-item bg-primary/10 font-medium text-foreground"
            : "text-foreground/70 hover:bg-foreground/5 hover:text-foreground",
        )}
      >
        <a
          href={item.href}
          className="flex flex-1 items-center gap-2 truncate"
          data-testid={`nav-link-${item.id}`}
          onClick={() => {
            // Auto-expand sub-menu when navigating to parent.
            if (hasChildren && !isExpanded) onToggleExpand();
          }}
        >
          <Icon className={cn("h-[18px] w-[18px] shrink-0", isActive ? "text-primary" : "text-foreground/50 group-hover:text-foreground/70")} />
          <span className="truncate">{item.label}</span>
        </a>
        {hasChildren && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onToggleExpand();
            }}
            className="shrink-0 rounded p-0.5 text-foreground/40 transition-colors hover:bg-foreground/10 hover:text-foreground"
            aria-label={isExpanded ? "Collapse" : "Expand"}
            data-testid={`nav-toggle-${item.id}`}
          >
            {isExpanded ? (
              <ChevronDown className="h-3.5 w-3.5" />
            ) : (
              <ChevronRight className="h-3.5 w-3.5" />
            )}
          </button>
        )}
      </div>
      {hasChildren && (
        <SubMenuPanel expanded={isExpanded}>
          {item.children!.map((child) => (
            <SubMenuItem key={child.id} item={child} currentPath={currentPath} />
          ))}
        </SubMenuPanel>
      )}
    </li>
  );
}

/** Collapsed sidebar item — icon only with tooltip. */
function CollapsedItem({ item, currentPath }: { item: SidebarNavItem; currentPath?: string }): React.ReactNode {
  const isActive =
    currentPath === item.href ||
    (item.children?.some((c) => currentPath === c.href || (currentPath?.startsWith(c.href + "/") ?? false)) ?? false);
  const Icon = item.icon;

  return (
    <li data-testid={`nav-item-${item.id}`}>
      <a
        href={item.href}
        aria-label={item.label}
        title={item.label}
        data-testid={`nav-link-${item.id}`}
        className={cn(
          "flex items-center justify-center rounded-md p-2 transition-colors duration-150",
          isActive
            ? "bg-primary/10 text-primary"
            : "text-foreground/50 hover:bg-foreground/5 hover:text-foreground",
        )}
      >
        <Icon className="h-[18px] w-[18px]" />
      </a>
    </li>
  );
}

/** Sidebar with tier-filtered navigation, collapsible sub-menus, and visual polish. */
export function Sidebar({
  items,
  currentPath,
  collapsed = false,
  onToggle,
}: SidebarProps): React.ReactNode {
  const activeParentId = findActiveParentId(items, currentPath);
  const [expandedId, setExpandedId] = useState<string | null>(activeParentId);

  // Derive whether the expanded section should be overridden by route.
  // If the current route falls under a parent that isn't expanded, auto-expand it.
  const routeParentId = findActiveParentId(items, currentPath);
  const effectiveExpandedId = routeParentId && routeParentId !== expandedId ? routeParentId : expandedId;

  const handleToggle = (id: string): void => {
    setExpandedId((prev) => (prev === id ? null : id));
  };

  return (
    <aside
      data-testid="sidebar"
      className={cn(
        "sidebar-root flex h-full flex-col border-r border-foreground/10 bg-sidebar-bg transition-all duration-200",
        collapsed ? "w-12" : "w-64",
      )}
    >
      <div className="flex items-center justify-between px-3 py-3">
        {!collapsed && (
          <div className="flex items-center gap-2">
            <LayoutDashboard className="h-5 w-5 text-primary" />
            <span className="text-sm font-semibold text-foreground">DEFT</span>
          </div>
        )}
        <button
          onClick={onToggle}
          aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          className="rounded-md p-1.5 text-foreground/50 transition-colors hover:bg-foreground/10 hover:text-foreground"
          data-testid="sidebar-toggle"
        >
          {collapsed ? <PanelLeft className="h-4 w-4" /> : <PanelLeftClose className="h-4 w-4" />}
        </button>
      </div>

      {collapsed ? (
        <nav className="flex-1 overflow-y-auto px-1 pb-4" aria-label="Sidebar navigation">
          <ul role="tree" className="flex flex-col gap-1">
            {items.map((item) => (
              <CollapsedItem key={item.id} item={item} currentPath={currentPath} />
            ))}
          </ul>
        </nav>
      ) : (
        <nav className="flex-1 overflow-y-auto px-2 pb-4" aria-label="Sidebar navigation">
          <ul role="tree" className="flex flex-col gap-0.5">
            {items.map((item) => (
              <TopLevelItem
                key={item.id}
                item={item}
                currentPath={currentPath}
                isExpanded={effectiveExpandedId === item.id}
                onToggleExpand={() => handleToggle(item.id)}
              />
            ))}
          </ul>
        </nav>
      )}
    </aside>
  );
}
