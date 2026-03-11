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

/** Navigation item used in the sidebar tree. */
export interface NavItem {
  id: string;
  label: string;
  href: string;
  type: "org" | "space" | "board";
  children?: NavItem[];
}

interface SidebarProps {
  items: NavItem[];
  currentPath?: string;
  collapsed?: boolean;
  onToggle?: () => void;
}

function NavNode({
  item,
  currentPath,
  depth,
}: {
  item: NavItem;
  currentPath?: string;
  depth: number;
}): React.ReactNode {
  const [expanded, setExpanded] = useState(true);
  const hasChildren = item.children && item.children.length > 0;
  const isActive = currentPath === item.href;

  return (
    <li data-testid={`nav-item-${item.id}`}>
      <div
        className={cn(
          "flex items-center gap-1 rounded-md px-2 py-1.5 text-sm transition-colors",
          isActive
            ? "bg-foreground/10 font-medium text-foreground"
            : "text-foreground/70 hover:bg-foreground/5 hover:text-foreground",
        )}
        style={{ paddingLeft: `${depth * 12 + 8}px` }}
      >
        {hasChildren ? (
          <button
            onClick={() => setExpanded(!expanded)}
            className="shrink-0 rounded p-0.5 hover:bg-foreground/10"
            aria-label={expanded ? "Collapse" : "Expand"}
            data-testid={`nav-toggle-${item.id}`}
          >
            {expanded ? (
              <ChevronDown className="h-3.5 w-3.5" />
            ) : (
              <ChevronRight className="h-3.5 w-3.5" />
            )}
          </button>
        ) : (
          <span className="w-4.5 shrink-0" />
        )}
        <a href={item.href} className="flex-1 truncate" data-testid={`nav-link-${item.id}`}>
          {item.label}
        </a>
      </div>
      {hasChildren && expanded && (
        <ul role="group">
          {item.children?.map((child) => (
            <NavNode key={child.id} item={child} currentPath={currentPath} depth={depth + 1} />
          ))}
        </ul>
      )}
    </li>
  );
}

/** Sidebar with collapsible org → space → board navigation tree. */
export function Sidebar({
  items,
  currentPath,
  collapsed = false,
  onToggle,
}: SidebarProps): React.ReactNode {
  return (
    <aside
      data-testid="sidebar"
      className={cn(
        "flex h-full flex-col border-r border-foreground/10 bg-background transition-all duration-200",
        collapsed ? "w-12" : "w-64",
      )}
    >
      <div className="flex items-center justify-between px-3 py-3">
        {!collapsed && (
          <div className="flex items-center gap-2">
            <LayoutDashboard className="h-5 w-5 text-foreground/70" />
            <span className="text-sm font-semibold text-foreground">DEFT</span>
          </div>
        )}
        <button
          onClick={onToggle}
          aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          className="rounded-md p-1.5 text-foreground/50 hover:bg-foreground/10 hover:text-foreground"
          data-testid="sidebar-toggle"
        >
          {collapsed ? <PanelLeft className="h-4 w-4" /> : <PanelLeftClose className="h-4 w-4" />}
        </button>
      </div>
      {!collapsed && (
        <nav className="flex-1 overflow-y-auto px-2 pb-4" aria-label="Sidebar navigation">
          <ul role="tree">
            {items.map((item) => (
              <NavNode key={item.id} item={item} currentPath={currentPath} depth={0} />
            ))}
          </ul>
        </nav>
      )}
    </aside>
  );
}
