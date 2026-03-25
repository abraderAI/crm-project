"use client";

import { User, DollarSign, TrendingUp, Calendar, AlertCircle } from "lucide-react";
import { cn } from "@/lib/utils";
import type { LeadCard } from "@/lib/crm-types";
import { formatCurrency, isOverdue, STAGE_COLORS } from "@/lib/crm-types";

export interface KanbanCardProps {
  card: LeadCard;
  onClick?: (threadId: string) => void;
  href?: string;
  isDragging?: boolean;
  /** Resolve a user ID to a display label. */
  formatUser?: (userId: string) => string;
}

/** Single lead card displayed in a Kanban column. */
export function KanbanCard({
  card,
  onClick,
  href,
  isDragging = false,
  formatUser,
}: KanbanCardProps): React.ReactNode {
  const { thread, lead } = card;

  const content = (
    <div
      data-testid={`kanban-card-${thread.id}`}
      draggable
      onDragStart={(e) => {
        e.dataTransfer.setData("text/plain", thread.id);
        e.dataTransfer.effectAllowed = "move";
      }}
      className={cn(
        "rounded-md border border-border bg-background p-3 shadow-sm transition-shadow",
        "cursor-grab hover:shadow-md active:cursor-grabbing",
        isDragging && "opacity-50 shadow-lg",
      )}
      onClick={onClick ? () => onClick(thread.id) : undefined}
      role={onClick ? "button" : undefined}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={
        onClick
          ? (e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                onClick(thread.id);
              }
            }
          : undefined
      }
    >
      {/* Title */}
      <h4
        className="truncate text-sm font-medium text-foreground"
        data-testid={`kanban-card-title-${thread.id}`}
      >
        {thread.title}
      </h4>

      {/* Company */}
      {lead.company && (
        <p
          className="mt-1 truncate text-xs text-muted-foreground"
          data-testid={`kanban-card-company-${thread.id}`}
        >
          {lead.company}
        </p>
      )}

      {/* Metadata row */}
      <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
        {lead.deal_amount != null && lead.deal_amount > 0 ? (
          <span
            className="flex items-center gap-0.5 font-medium text-foreground"
            data-testid={`kanban-card-deal-${thread.id}`}
          >
            <DollarSign className="h-3 w-3" />
            {formatCurrency(lead.deal_amount)}
          </span>
        ) : lead.value != null && lead.value > 0 ? (
          <span
            className="flex items-center gap-0.5"
            data-testid={`kanban-card-value-${thread.id}`}
          >
            <DollarSign className="h-3 w-3" />
            {formatCurrency(lead.value)}
          </span>
        ) : null}
        {lead.weighted_forecast != null && lead.weighted_forecast > 0 && (
          <span
            className="flex items-center gap-0.5 text-green-600 dark:text-green-400"
            data-testid={`kanban-card-forecast-${thread.id}`}
          >
            <TrendingUp className="h-3 w-3" />
            {formatCurrency(lead.weighted_forecast)}
          </span>
        )}
        {lead.score != null && (
          <span
            className="flex items-center gap-0.5"
            data-testid={`kanban-card-score-${thread.id}`}
          >
            <TrendingUp className="h-3 w-3" />
            {lead.score}
          </span>
        )}
      </div>

      {/* Close date + overdue badge */}
      {lead.expected_close_date && (
        <div
          className="mt-1 flex items-center gap-1 text-xs"
          data-testid={`kanban-card-close-date-${thread.id}`}
        >
          <Calendar className="h-3 w-3 text-muted-foreground" />
          <span
            className={cn(
              isOverdue(lead.expected_close_date)
                ? "font-medium text-red-600 dark:text-red-400"
                : "text-muted-foreground",
            )}
          >
            {new Date(lead.expected_close_date).toLocaleDateString()}
          </span>
          {isOverdue(lead.expected_close_date) && (
            <span
              className="inline-flex items-center gap-0.5 rounded bg-red-100 px-1.5 py-0.5 text-xs font-medium text-red-700 dark:bg-red-900/30 dark:text-red-300"
              data-testid={`kanban-card-overdue-${thread.id}`}
            >
              <AlertCircle className="h-3 w-3" />
              Overdue
            </span>
          )}
        </div>
      )}

      {/* Assigned to */}
      {lead.assigned_to && (
        <div
          className="mt-2 flex items-center gap-1 text-xs text-muted-foreground"
          data-testid={`kanban-card-assignee-${thread.id}`}
        >
          <User className="h-3 w-3" />
          <span className="truncate">
            {formatUser ? formatUser(lead.assigned_to) : lead.assigned_to}
          </span>
        </div>
      )}
    </div>
  );

  if (href) {
    return (
      <a href={href} className="block no-underline" data-testid={`kanban-card-link-${thread.id}`}>
        {content}
      </a>
    );
  }

  return content;
}

/** Stage badge for column headers. */
export function StageBadge({ stage, count }: { stage: string; count: number }): React.ReactNode {
  const colorClass =
    STAGE_COLORS[stage as keyof typeof STAGE_COLORS] ?? "bg-gray-100 text-gray-800";
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
        colorClass,
      )}
      data-testid={`stage-badge-${stage}`}
    >
      {count}
    </span>
  );
}
