import Link from "next/link";
import {
  Building2,
  FileBarChart,
  Headset,
  ScrollText,
  Settings,
  ToggleRight,
  TrendingUp,
  Users,
} from "lucide-react";
import { fetchAdminSalesMetrics, fetchAdminSupportMetrics } from "@/lib/admin-api";
import { MetricCard } from "@/components/reports/metric-card";

/** Quick-link definition for the admin launchpad. */
interface QuickLink {
  label: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
}

const QUICK_LINKS: QuickLink[] = [
  { label: "Organizations", href: "/admin/orgs", icon: Building2 },
  { label: "Users", href: "/admin/users", icon: Users },
  { label: "Audit Log", href: "/admin/audit-log", icon: ScrollText },
  { label: "Feature Flags", href: "/admin/feature-flags", icon: ToggleRight },
  { label: "Settings", href: "/admin/settings", icon: Settings },
];

export default async function AdminOverviewPage(): Promise<React.ReactNode> {
  const [supportMetrics, salesMetrics] = await Promise.all([
    fetchAdminSupportMetrics(),
    fetchAdminSalesMetrics(),
  ]);

  const openTickets = supportMetrics.status_breakdown?.open ?? 0;
  const assignedTickets = supportMetrics.status_breakdown?.assigned ?? 0;
  const avgResolution = supportMetrics.avg_resolution_hours;

  const totalLeads = salesMetrics.pipeline_funnel?.reduce((sum, s) => sum + s.count, 0) ?? 0;
  const winRate = salesMetrics.win_rate ?? 0;
  const avgDealValue = salesMetrics.avg_deal_value;

  return (
    <div data-testid="admin-overview" className="flex flex-col gap-6">
      {/* Support snapshot */}
      <section>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <Headset className="h-4 w-4" />
            Support
          </h2>
          <Link
            href="/admin/reports/support"
            className="text-xs text-muted-foreground hover:text-foreground"
            data-testid="link-support-report"
          >
            View full report →
          </Link>
        </div>
        <div className="grid gap-4 sm:grid-cols-3" data-testid="support-snapshot">
          <MetricCard label="Open Tickets" value={openTickets} href="/admin/reports/support" />
          <MetricCard
            label="Assigned Tickets"
            value={assignedTickets}
            href="/admin/reports/support"
          />
          <MetricCard
            label="Avg Resolution"
            value={avgResolution != null ? `${avgResolution.toFixed(1)} hrs` : "–"}
            href="/admin/reports/support"
          />
        </div>
      </section>

      {/* Sales snapshot */}
      <section>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <TrendingUp className="h-4 w-4" />
            Sales
          </h2>
          <Link
            href="/admin/reports/sales"
            className="text-xs text-muted-foreground hover:text-foreground"
            data-testid="link-sales-report"
          >
            View full report →
          </Link>
        </div>
        <div className="grid gap-4 sm:grid-cols-3" data-testid="sales-snapshot">
          <MetricCard label="Total Leads" value={totalLeads} href="/admin/reports/sales" />
          <MetricCard
            label="Win Rate"
            value={`${(winRate * 100).toFixed(1)}%`}
            href="/admin/reports/sales"
          />
          <MetricCard
            label="Avg Deal Value"
            value={avgDealValue != null ? `$${avgDealValue.toLocaleString()}` : "–"}
            href="/admin/reports/sales"
          />
        </div>
      </section>

      {/* Quick links */}
      <section>
        <h2 className="mb-3 text-sm font-semibold text-foreground">
          <FileBarChart className="mr-2 inline h-4 w-4" />
          Quick Links
        </h2>
        <div className="flex flex-wrap gap-3" data-testid="admin-quick-links">
          {QUICK_LINKS.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className="flex items-center gap-2 rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-accent/50"
              data-testid={`quick-link-${link.label.toLowerCase().replace(/\s+/g, "-")}`}
            >
              <link.icon className="h-3.5 w-3.5" />
              {link.label}
            </Link>
          ))}
        </div>
      </section>
    </div>
  );
}
