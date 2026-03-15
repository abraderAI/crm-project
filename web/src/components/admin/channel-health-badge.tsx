"use client";

import { cn } from "@/lib/utils";

export interface ChannelHealthBadgeProps {
  /** Health status string: "healthy" | "degraded" | "error" | "unconfigured" or any other value. */
  status: string;
}

/** Map health status to badge colour classes. */
function statusColor(status: string): string {
  switch (status.toLowerCase()) {
    case "healthy":
      return "bg-green-100 text-green-800";
    case "degraded":
      return "bg-yellow-100 text-yellow-800";
    case "error":
      return "bg-red-100 text-red-800";
    case "unconfigured":
      return "bg-gray-100 text-gray-600";
    default:
      return "bg-muted text-muted-foreground";
  }
}

/** Small reusable badge component colour-coded by channel health status. */
export function ChannelHealthBadge({ status }: ChannelHealthBadgeProps): React.ReactNode {
  return (
    <span
      data-testid="channel-health-badge"
      className={cn(
        "inline-block rounded-full px-2.5 py-0.5 text-xs font-medium",
        statusColor(status),
      )}
    >
      {status}
    </span>
  );
}
