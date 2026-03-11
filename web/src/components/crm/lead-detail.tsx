"use client";

import { Building2, Mail, Globe, ExternalLink } from "lucide-react";
import type { Thread, Message } from "@/lib/api-types";
import type { LeadData, ScoreBreakdown, PipelineStage } from "@/lib/crm-types";
import {
  parseLeadData,
  resolveStage,
  STAGE_LABELS,
  STAGE_COLORS,
  parseEnrichmentData,
} from "@/lib/crm-types";
import { cn } from "@/lib/utils";
import { MessageTimeline } from "@/components/thread/message-timeline";
import { ScoreBreakdownView } from "./score-breakdown";
import { EnrichmentSection } from "./enrichment-section";

export interface LeadDetailProps {
  thread: Thread;
  messages: Message[];
  scoreBreakdown?: ScoreBreakdown | null;
  onEnrich?: () => void;
  enrichLoading?: boolean;
  onEditMessage?: (messageId: string) => void;
  currentUserId?: string;
  /** Link to the provisioned customer org (for closed_won). */
  customerOrgHref?: string;
}

/** Full lead detail view: metadata sidebar, activity timeline, score, enrichment. */
export function LeadDetail({
  thread,
  messages,
  scoreBreakdown,
  onEnrich,
  enrichLoading = false,
  onEditMessage,
  currentUserId,
  customerOrgHref,
}: LeadDetailProps): React.ReactNode {
  const lead = parseLeadData(thread.metadata);
  const stage = resolveStage(thread);
  const enrichment = parseEnrichmentData(thread.metadata);

  return (
    <div className="flex gap-6" data-testid="lead-detail">
      {/* Main content area */}
      <div className="min-w-0 flex-1 space-y-6">
        {/* Header */}
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-bold text-foreground" data-testid="lead-detail-title">
              {thread.title}
            </h1>
            <StagePill stage={stage} />
          </div>
          {thread.body && (
            <p className="mt-2 text-sm text-muted-foreground" data-testid="lead-detail-body">
              {thread.body}
            </p>
          )}
        </div>

        {/* Customer org link (closed_won) */}
        {customerOrgHref && stage === "closed_won" && (
          <a
            href={customerOrgHref}
            className="inline-flex items-center gap-1 rounded-md bg-green-100 px-3 py-1.5 text-sm font-medium text-green-800 hover:bg-green-200 dark:bg-green-900/30 dark:text-green-300"
            data-testid="lead-detail-customer-link"
          >
            <ExternalLink className="h-3.5 w-3.5" />
            View customer organization
          </a>
        )}

        {/* AI Enrichment */}
        <div className="rounded-lg border border-border p-4">
          <EnrichmentSection enrichment={enrichment} onEnrich={onEnrich} loading={enrichLoading} />
        </div>

        {/* Score breakdown */}
        {scoreBreakdown && (
          <div className="rounded-lg border border-border p-4">
            <ScoreBreakdownView breakdown={scoreBreakdown} />
          </div>
        )}

        {/* Activity timeline */}
        <div>
          <h2 className="mb-3 text-sm font-semibold text-foreground">
            Activity ({messages.length})
          </h2>
          <MessageTimeline
            messages={messages}
            currentUserId={currentUserId}
            onEdit={onEditMessage}
          />
        </div>
      </div>

      {/* Sidebar */}
      <LeadSidebar lead={lead} stage={stage} thread={thread} />
    </div>
  );
}

/** Stage pill badge. */
function StagePill({ stage }: { stage: PipelineStage }): React.ReactNode {
  const colorClass = STAGE_COLORS[stage];
  return (
    <span
      className={cn("rounded-full px-2.5 py-0.5 text-xs font-medium", colorClass)}
      data-testid="lead-detail-stage"
    >
      {STAGE_LABELS[stage]}
    </span>
  );
}

interface LeadSidebarProps {
  lead: LeadData;
  stage: PipelineStage;
  thread: Thread;
}

/** Sidebar displaying lead metadata details. */
function LeadSidebar({ lead, stage, thread }: LeadSidebarProps): React.ReactNode {
  return (
    <aside className="w-64 shrink-0 space-y-4" data-testid="lead-sidebar">
      {/* Company info */}
      <div className="rounded-lg border border-border p-4">
        <h3 className="mb-3 text-sm font-semibold text-foreground">Lead Information</h3>
        <div className="space-y-2">
          {lead.company && (
            <SidebarField
              icon={Building2}
              label="Company"
              value={lead.company}
              testId="lead-sidebar-company"
            />
          )}
          {lead.contact_name && (
            <SidebarField label="Contact" value={lead.contact_name} testId="lead-sidebar-contact" />
          )}
          {lead.contact_email && (
            <SidebarField
              icon={Mail}
              label="Email"
              value={lead.contact_email}
              testId="lead-sidebar-email"
            />
          )}
          {lead.source && (
            <SidebarField
              icon={Globe}
              label="Source"
              value={lead.source}
              testId="lead-sidebar-source"
            />
          )}
          {lead.assigned_to && (
            <SidebarField
              label="Assigned to"
              value={lead.assigned_to}
              testId="lead-sidebar-assignee"
            />
          )}
          {lead.value != null && (
            <SidebarField
              label="Deal value"
              value={`$${lead.value.toLocaleString()}`}
              testId="lead-sidebar-value"
            />
          )}
          {lead.score != null && (
            <SidebarField label="Score" value={String(lead.score)} testId="lead-sidebar-score" />
          )}
          <SidebarField label="Stage" value={STAGE_LABELS[stage]} testId="lead-sidebar-stage" />
        </div>
      </div>

      {/* Thread metadata */}
      <div className="rounded-lg border border-border p-4">
        <h3 className="mb-3 text-sm font-semibold text-foreground">Thread Details</h3>
        <div className="space-y-2">
          <SidebarField label="Status" value={thread.status ?? "—"} testId="lead-sidebar-status" />
          <SidebarField
            label="Priority"
            value={thread.priority ?? "—"}
            testId="lead-sidebar-priority"
          />
          <SidebarField
            label="Votes"
            value={String(thread.vote_score)}
            testId="lead-sidebar-votes"
          />
          {thread.is_pinned && (
            <SidebarField label="Pinned" value="Yes" testId="lead-sidebar-pinned" />
          )}
          {thread.is_locked && (
            <SidebarField label="Locked" value="Yes" testId="lead-sidebar-locked" />
          )}
        </div>
      </div>
    </aside>
  );
}

interface SidebarFieldProps {
  icon?: typeof Building2;
  label: string;
  value: string;
  testId: string;
}

function SidebarField({ icon: Icon, label, value, testId }: SidebarFieldProps): React.ReactNode {
  return (
    <div className="flex items-center justify-between" data-testid={testId}>
      <span className="flex items-center gap-1 text-xs text-muted-foreground">
        {Icon && <Icon className="h-3 w-3" />}
        {label}
      </span>
      <span className="text-xs font-medium text-foreground">{value}</span>
    </div>
  );
}
