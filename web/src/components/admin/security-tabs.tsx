"use client";

import { useState } from "react";
import { ShieldAlert } from "lucide-react";
import { cn } from "@/lib/utils";
import { SecurityLog } from "./security-log";
import type { SecurityLogEntry } from "@/lib/api-types";

type TabKey = "recent-logins" | "failed-auths";

const TABS: { key: TabKey; label: string }[] = [
  { key: "recent-logins", label: "Recent Logins" },
  { key: "failed-auths", label: "Failed Auths" },
];

export interface SecurityTabsProps {
  /** Recent login entries. */
  recentLogins: SecurityLogEntry[];
  /** Failed authentication entries. */
  failedAuths: SecurityLogEntry[];
  /** Whether more recent logins are available. */
  loginsHasMore: boolean;
  /** Whether more failed auths are available. */
  failedHasMore: boolean;
}

/** Security monitoring page with tabbed Recent Logins and Failed Auths views. */
export function SecurityTabs({
  recentLogins,
  failedAuths,
  loginsHasMore,
  failedHasMore,
}: SecurityTabsProps): React.ReactNode {
  const [activeTab, setActiveTab] = useState<TabKey>("recent-logins");

  return (
    <div data-testid="security-tabs" className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <ShieldAlert className="h-5 w-5 text-muted-foreground" data-testid="security-icon" />
        <h2 className="text-lg font-semibold text-foreground">Security Monitoring</h2>
      </div>

      <div className="flex gap-1 border-b border-border" role="tablist">
        {TABS.map(({ key, label }) => {
          const isActive = activeTab === key;
          return (
            <button
              key={key}
              role="tab"
              aria-selected={isActive}
              data-testid={`tab-${key}`}
              onClick={() => setActiveTab(key)}
              className={cn(
                "border-b-2 px-4 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "border-primary text-foreground"
                  : "border-transparent text-muted-foreground hover:border-border hover:text-foreground",
              )}
            >
              {label}
            </button>
          );
        })}
      </div>

      <div role="tabpanel">
        {activeTab === "recent-logins" && (
          <SecurityLog entries={recentLogins} hasMore={loginsHasMore} />
        )}
        {activeTab === "failed-auths" && (
          <SecurityLog entries={failedAuths} hasMore={failedHasMore} />
        )}
      </div>
    </div>
  );
}
