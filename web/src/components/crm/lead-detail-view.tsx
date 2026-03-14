"use client";

import { useAuth } from "@clerk/nextjs";

import type { Thread, Message } from "@/lib/api-types";
import type { ScoreBreakdown } from "@/lib/crm-types";
import { LeadDetail } from "./lead-detail";

export interface LeadDetailViewProps {
  /** Thread representing the lead. */
  thread: Thread;
  /** Messages for activity timeline. */
  messages: Message[];
  /** Optional score breakdown. */
  scoreBreakdown?: ScoreBreakdown | null;
  /** Link to provisioned customer org (for closed_won). */
  customerOrgHref?: string;
}

/** Client wrapper wiring LeadDetail with Clerk auth for currentUserId. */
export function LeadDetailView({
  thread,
  messages,
  scoreBreakdown,
  customerOrgHref,
}: LeadDetailViewProps): React.ReactNode {
  const { userId } = useAuth();

  return (
    <LeadDetail
      thread={thread}
      messages={messages}
      scoreBreakdown={scoreBreakdown}
      currentUserId={userId ?? undefined}
      customerOrgHref={customerOrgHref}
    />
  );
}
