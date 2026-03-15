"use client";

import { Mail, Phone, MessageSquare, Settings, AlertTriangle } from "lucide-react";
import { cn } from "@/lib/utils";
import { ChannelHealthBadge } from "./channel-health-badge";
import type { ChannelHealth, ChannelType } from "@/lib/api-types";

/** Channel metadata for display. */
interface ChannelMeta {
  type: ChannelType;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
}

const CHANNELS: ChannelMeta[] = [
  { type: "email", label: "Email", icon: Mail },
  { type: "voice", label: "Voice", icon: Phone },
  { type: "chat", label: "Chat", icon: MessageSquare },
];

export interface ChannelOverviewProps {
  /** Health data for each channel type, keyed by channel type. */
  healthMap: Record<ChannelType, ChannelHealth | null>;
  /** DLQ failed event counts per channel type. */
  dlqCounts?: Record<ChannelType, number>;
  /** Whether data is loading. */
  loading?: boolean;
}

/** Format a date string for display. */
function formatTime(dateStr: string | undefined | null): string {
  if (!dateStr) return "Never";
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return "Never";
    return d.toLocaleString("en-US", {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return "Never";
  }
}

/** Channel overview — grid of 3 cards, one per channel type. */
export function ChannelOverview({
  healthMap,
  dlqCounts = { email: 0, voice: 0, chat: 0 },
  loading = false,
}: ChannelOverviewProps): React.ReactNode {
  return (
    <div data-testid="channel-overview" className="flex flex-col gap-4">
      <h2 className="text-lg font-semibold text-foreground">IO Channels</h2>

      {loading && (
        <div
          className="py-8 text-center text-sm text-muted-foreground"
          data-testid="channel-overview-loading"
        >
          Loading channel data...
        </div>
      )}

      {!loading && (
        <div className="grid gap-4 sm:grid-cols-3" data-testid="channel-grid">
          {CHANNELS.map(({ type, label, icon: Icon }) => {
            const health = healthMap[type];
            const failedCount = dlqCounts[type] ?? 0;
            const status = health?.status ?? "unconfigured";
            const enabled = health?.enabled ?? false;

            return (
              <div
                key={type}
                className="rounded-lg border border-border p-4"
                data-testid={`channel-card-${type}`}
              >
                <div className="flex items-center gap-2">
                  <Icon className="h-5 w-5 text-muted-foreground" />
                  <h3 className="text-sm font-semibold text-foreground">{label}</h3>
                  <span
                    className={cn(
                      "ml-auto rounded-full px-2 py-0.5 text-xs font-medium",
                      enabled ? "bg-green-100 text-green-800" : "bg-muted text-muted-foreground",
                    )}
                    data-testid={`channel-enabled-${type}`}
                  >
                    {enabled ? "Enabled" : "Disabled"}
                  </span>
                </div>

                <div className="mt-3 flex flex-col gap-2">
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-muted-foreground">Health:</span>
                    <ChannelHealthBadge status={status} />
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-muted-foreground">Last event:</span>
                    <span
                      className="text-xs text-foreground"
                      data-testid={`channel-last-event-${type}`}
                    >
                      {formatTime(health?.last_event_at)}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-muted-foreground">Error rate:</span>
                    <span
                      className="text-xs text-foreground"
                      data-testid={`channel-error-rate-${type}`}
                    >
                      {health ? `${(health.error_rate * 100).toFixed(1)}%` : "N/A"}
                    </span>
                  </div>
                </div>

                <div className="mt-4 flex items-center gap-2">
                  <a
                    href={`/admin/channels/${type}`}
                    data-testid={`channel-configure-${type}`}
                    className="inline-flex items-center gap-1 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90"
                  >
                    <Settings className="h-3 w-3" />
                    Configure
                  </a>
                  <a
                    href={`/admin/channels/${type}#dlq`}
                    data-testid={`channel-dlq-${type}`}
                    className="inline-flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-accent"
                  >
                    <AlertTriangle className="h-3 w-3" />
                    View DLQ
                    {failedCount > 0 && (
                      <span
                        className="ml-1 rounded-full bg-red-500 px-1.5 py-0.5 text-[10px] font-bold text-white"
                        data-testid={`channel-dlq-count-${type}`}
                      >
                        {failedCount}
                      </span>
                    )}
                  </a>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
