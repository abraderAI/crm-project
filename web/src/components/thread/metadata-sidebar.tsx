"use client";

import { parseMetadata } from "@/lib/utils";

export interface MetadataSidebarProps {
  status?: string;
  priority?: string;
  stage?: string;
  assignedTo?: string;
  voteScore: number;
  /** Raw metadata JSON string or parsed object. */
  metadata?: string | Record<string, unknown>;
  isPinned?: boolean;
  isLocked?: boolean;
}

interface FieldRowProps {
  label: string;
  value: string | number;
  testId: string;
}

function FieldRow({ label, value, testId }: FieldRowProps): React.ReactNode {
  return (
    <div className="flex items-center justify-between py-1.5" data-testid={testId}>
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-xs font-medium text-foreground">{value}</span>
    </div>
  );
}

/** Sidebar displaying thread metadata: status, priority, stage, assigned_to, custom fields. */
export function MetadataSidebar({
  status,
  priority,
  stage,
  assignedTo,
  voteScore,
  metadata,
  isPinned,
  isLocked,
}: MetadataSidebarProps): React.ReactNode {
  const parsed = typeof metadata === "string" ? parseMetadata(metadata) : (metadata ?? {});
  const customFields = Object.entries(parsed).filter(
    ([key]) => !["status", "priority", "stage", "assigned_to"].includes(key),
  );

  return (
    <aside
      className="w-64 rounded-lg border border-border bg-background p-4"
      data-testid="metadata-sidebar"
    >
      <h3 className="mb-3 text-sm font-semibold text-foreground">Details</h3>
      <div className="divide-y divide-border">
        {/* Core fields */}
        <div className="pb-2">
          {status && <FieldRow label="Status" value={status} testId="sidebar-status" />}
          {priority && <FieldRow label="Priority" value={priority} testId="sidebar-priority" />}
          {stage && <FieldRow label="Stage" value={stage} testId="sidebar-stage" />}
          {assignedTo && (
            <FieldRow label="Assigned to" value={assignedTo} testId="sidebar-assigned" />
          )}
          <FieldRow label="Votes" value={voteScore} testId="sidebar-votes" />
        </div>

        {/* Status flags */}
        {(isPinned || isLocked) && (
          <div className="py-2">
            {isPinned && <FieldRow label="Pinned" value="Yes" testId="sidebar-pinned" />}
            {isLocked && <FieldRow label="Locked" value="Yes" testId="sidebar-locked" />}
          </div>
        )}

        {/* Custom metadata */}
        {customFields.length > 0 && (
          <div className="pt-2">
            <p className="mb-1 text-xs font-medium text-muted-foreground">Custom fields</p>
            {customFields.map(([key, value]) => (
              <FieldRow
                key={key}
                label={key}
                value={String(value)}
                testId={`sidebar-custom-${key}`}
              />
            ))}
          </div>
        )}
      </div>
    </aside>
  );
}
