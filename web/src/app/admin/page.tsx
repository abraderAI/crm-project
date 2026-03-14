import {
  Building2,
  Users,
  MessageSquare,
  FileText,
  AlertTriangle,
  Bell,
  Database,
} from "lucide-react";
import { fetchAdminStats } from "@/lib/admin-api";
import type { CountStats } from "@/lib/api-types";

/** Format bytes into a human-readable string. */
function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / 1024 ** i).toFixed(1)} ${units[i]}`;
}

function StatCard({
  label,
  stats,
  icon: Icon,
}: {
  label: string;
  stats: CountStats;
  icon: React.ComponentType<{ className?: string }>;
}): React.ReactNode {
  return (
    <div
      className="rounded-lg border border-border p-4"
      data-testid={`stat-card-${label.toLowerCase()}`}
    >
      <div className="flex items-center gap-2">
        <Icon className="h-5 w-5 text-muted-foreground" />
        <p className="text-sm font-medium text-muted-foreground">{label}</p>
      </div>
      <p className="mt-2 text-3xl font-bold text-foreground">{stats.total.toLocaleString()}</p>
      <div className="mt-1 flex gap-3 text-xs text-muted-foreground">
        <span>7d: +{stats.last_7d.toLocaleString()}</span>
        <span>30d: +{stats.last_30d.toLocaleString()}</span>
      </div>
    </div>
  );
}

export default async function AdminOverviewPage(): Promise<React.ReactNode> {
  const stats = await fetchAdminStats();

  return (
    <div data-testid="admin-overview" className="flex flex-col gap-6">
      {/* Entity stats */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label="Organizations" stats={stats.orgs} icon={Building2} />
        <StatCard label="Users" stats={stats.users} icon={Users} />
        <StatCard label="Threads" stats={stats.threads} icon={FileText} />
        <StatCard label="Messages" stats={stats.messages} icon={MessageSquare} />
      </div>

      {/* System health */}
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="rounded-lg border border-border p-4" data-testid="stat-db-size">
          <div className="flex items-center gap-2">
            <Database className="h-5 w-5 text-muted-foreground" />
            <p className="text-sm font-medium text-muted-foreground">Database Size</p>
          </div>
          <p className="mt-2 text-2xl font-bold text-foreground">
            {formatBytes(stats.db_size_bytes)}
          </p>
        </div>

        <div className="rounded-lg border border-border p-4" data-testid="stat-failed-webhooks">
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-muted-foreground" />
            <p className="text-sm font-medium text-muted-foreground">Failed Webhooks (24h)</p>
          </div>
          <p className="mt-2 text-2xl font-bold text-foreground">{stats.failed_webhooks_24h}</p>
        </div>

        <div
          className="rounded-lg border border-border p-4"
          data-testid="stat-pending-notifications"
        >
          <div className="flex items-center gap-2">
            <Bell className="h-5 w-5 text-muted-foreground" />
            <p className="text-sm font-medium text-muted-foreground">Pending Notifications</p>
          </div>
          <p className="mt-2 text-2xl font-bold text-foreground">{stats.pending_notifications}</p>
        </div>
      </div>
    </div>
  );
}
