import {
  Activity,
  BarChart3,
  Bell,
  BrainCircuit,
  Building2,
  CreditCard,
  FileBarChart,
  FileDown,
  Headset,
  Home,
  LayoutDashboard,
  MessageSquare,
  Plus,
  Radio,
  ScrollText,
  Search,
  Settings,
  Shield,
  ShieldAlert,
  ShieldCheck,
  ToggleRight,
  TrendingUp,
  UserCog,
  Users,
  Webhook,
  type LucideIcon,
} from "lucide-react";
import type { Tier } from "./tier-types";

/** A single navigation item with optional children (sub-menu). */
export interface SidebarNavItem {
  id: string;
  label: string;
  href: string;
  icon: LucideIcon;
  /** Minimum tier required to see this item. Defaults to 1 (visible to all). */
  minTier: Tier;
  /** Optional sub-menu items shown when this section is expanded. */
  children?: SidebarNavItem[];
  /** Optional badge text (e.g. unread count placeholder). */
  badge?: string;
}

/**
 * Full sidebar navigation tree.
 * Each top-level item may define children (sub-menus).
 * The `minTier` on each item controls visibility per user tier.
 */
export const SIDEBAR_NAV_ITEMS: SidebarNavItem[] = [
  {
    id: "home",
    label: "Home",
    href: "/",
    icon: Home,
    minTier: 1,
  },
  {
    id: "forum",
    label: "Forum",
    href: "/forum",
    icon: MessageSquare,
    minTier: 1,
  },
  {
    id: "docs",
    label: "Docs",
    href: "/docs",
    icon: LayoutDashboard,
    minTier: 1,
  },
  {
    id: "support",
    label: "Support",
    href: "/support",
    icon: Headset,
    minTier: 2,
    children: [
      {
        id: "support-tickets",
        label: "All Tickets",
        href: "/support",
        icon: Headset,
        minTier: 2,
      },
      {
        id: "support-new",
        label: "New Ticket",
        href: "/support/tickets/new",
        icon: Plus,
        minTier: 2,
      },
    ],
  },
  {
    id: "notifications",
    label: "Notifications",
    href: "/notifications",
    icon: Bell,
    minTier: 2,
  },
  {
    id: "search",
    label: "Search",
    href: "/search",
    icon: Search,
    minTier: 2,
  },
  {
    id: "settings",
    label: "Settings",
    href: "/settings",
    icon: Settings,
    minTier: 3,
  },
  {
    id: "crm",
    label: "CRM",
    href: "/crm",
    icon: BarChart3,
    minTier: 4,
    children: [
      {
        id: "crm-pipeline",
        label: "Pipeline",
        href: "/crm",
        icon: BarChart3,
        minTier: 4,
      },
      {
        id: "crm-leads",
        label: "Leads",
        href: "/crm/leads",
        icon: TrendingUp,
        minTier: 4,
      },
    ],
  },
  {
    id: "reports",
    label: "Reports",
    href: "/reports",
    icon: FileBarChart,
    minTier: 4,
    children: [
      {
        id: "reports-support",
        label: "Support Reports",
        href: "/reports/support",
        icon: Headset,
        minTier: 4,
      },
      {
        id: "reports-sales",
        label: "Sales Reports",
        href: "/reports/sales",
        icon: TrendingUp,
        minTier: 4,
      },
    ],
  },
  {
    id: "admin",
    label: "Admin",
    href: "/admin",
    icon: Shield,
    minTier: 6,
    children: [
      // Core
      {
        id: "admin-overview",
        label: "Overview",
        href: "/admin",
        icon: BarChart3,
        minTier: 6,
      },
      {
        id: "admin-orgs",
        label: "Organizations",
        href: "/admin/orgs",
        icon: Building2,
        minTier: 6,
      },
      {
        id: "admin-users",
        label: "Users",
        href: "/admin/users",
        icon: Users,
        minTier: 6,
      },
      {
        id: "admin-members",
        label: "Members",
        href: "/admin/members",
        icon: UserCog,
        minTier: 6,
      },
      // Security
      {
        id: "admin-audit-log",
        label: "Audit Log",
        href: "/admin/audit-log",
        icon: ScrollText,
        minTier: 6,
      },
      {
        id: "admin-security",
        label: "Security",
        href: "/admin/security",
        icon: ShieldAlert,
        minTier: 6,
      },
      {
        id: "admin-rbac",
        label: "RBAC Policy",
        href: "/admin/rbac-policy",
        icon: ShieldCheck,
        minTier: 6,
      },
      {
        id: "admin-moderation",
        label: "Moderation",
        href: "/admin/moderation",
        icon: Shield,
        minTier: 6,
      },
      {
        id: "admin-forums",
        label: "Forums",
        href: "/admin/forums",
        icon: MessageSquare,
        minTier: 6,
      },
      // Integrations
      {
        id: "admin-webhooks",
        label: "Webhooks",
        href: "/admin/webhooks",
        icon: Webhook,
        minTier: 6,
      },
      {
        id: "admin-channels",
        label: "Channels",
        href: "/admin/channels",
        icon: Radio,
        minTier: 6,
      },
      {
        id: "admin-feature-flags",
        label: "Feature Flags",
        href: "/admin/feature-flags",
        icon: ToggleRight,
        minTier: 6,
      },
      // Billing & Usage
      {
        id: "admin-billing",
        label: "Billing",
        href: "/admin/billing",
        icon: CreditCard,
        minTier: 6,
      },
      {
        id: "admin-api-usage",
        label: "API Usage",
        href: "/admin/api-usage",
        icon: Activity,
        minTier: 6,
      },
      {
        id: "admin-llm-usage",
        label: "LLM Usage",
        href: "/admin/llm-usage",
        icon: BrainCircuit,
        minTier: 6,
      },
      // Reports
      {
        id: "admin-reports-support",
        label: "Support Reports",
        href: "/admin/reports/support",
        icon: Headset,
        minTier: 6,
      },
      {
        id: "admin-reports-sales",
        label: "Sales Reports",
        href: "/admin/reports/sales",
        icon: TrendingUp,
        minTier: 6,
      },
      // System
      {
        id: "admin-exports",
        label: "Exports",
        href: "/admin/exports",
        icon: FileDown,
        minTier: 6,
      },
      {
        id: "admin-settings",
        label: "Settings",
        href: "/admin/settings",
        icon: Settings,
        minTier: 6,
      },
    ],
  },
];

/**
 * Filter navigation items by the user's tier.
 * Recursively filters children as well.
 */
export function getNavItemsForTier(tier: Tier): SidebarNavItem[] {
  return SIDEBAR_NAV_ITEMS.reduce<SidebarNavItem[]>((acc, item) => {
    if (item.minTier > tier) return acc;

    const filtered: SidebarNavItem = { ...item };
    if (item.children) {
      filtered.children = item.children.filter((child) => child.minTier <= tier);
      // Omit children array if empty after filtering.
      if (filtered.children.length === 0) {
        filtered.children = undefined;
      }
    }
    acc.push(filtered);
    return acc;
  }, []);
}
