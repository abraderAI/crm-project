import { ToggleRight } from "lucide-react";
import { fetchFeatureFlags } from "@/lib/admin-api";

/** Format a date string for display. */
function formatDate(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    return d.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
    });
  } catch {
    return dateStr;
  }
}

export default async function AdminFeatureFlagsPage(): Promise<React.ReactNode> {
  const flags = await fetchFeatureFlags();

  return (
    <div data-testid="admin-feature-flags" className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <ToggleRight className="h-5 w-5 text-muted-foreground" />
        <h2 className="text-lg font-semibold text-foreground">Feature Flags ({flags.length})</h2>
      </div>

      {flags.length === 0 ? (
        <p className="py-8 text-center text-sm text-muted-foreground" data-testid="flags-empty">
          No feature flags configured.
        </p>
      ) : (
        <div
          className="divide-y divide-border rounded-lg border border-border"
          data-testid="flag-list"
        >
          {flags.map((flag) => (
            <div
              key={flag.key}
              className="flex items-center gap-4 px-4 py-3"
              data-testid={`flag-row-${flag.key}`}
            >
              <span className="font-mono text-sm font-medium text-foreground">{flag.key}</span>

              <span
                className={
                  flag.enabled
                    ? "rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800"
                    : "rounded-full bg-muted px-2.5 py-0.5 text-xs font-medium text-muted-foreground"
                }
                data-testid={`flag-status-${flag.key}`}
              >
                {flag.enabled ? "Enabled" : "Disabled"}
              </span>

              {flag.org_scope && (
                <span className="text-xs text-muted-foreground">Org: {flag.org_scope}</span>
              )}

              <span className="ml-auto text-xs text-muted-foreground">
                Updated: {formatDate(flag.updated_at)}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
