import Link from "next/link";
import { ChevronRight } from "lucide-react";

import { fetchThread, fetchMessages } from "@/lib/user-api";
import { parseScoreBreakdown, parseLeadData } from "@/lib/crm-types";
import { LeadDetailView } from "@/components/crm/lead-detail-view";

interface LeadDetailPageProps {
  params: Promise<{ org: string; space: string; board: string; thread: string }>;
}

/**
 * CRM Lead Detail page — fetches a single lead thread with messages
 * and renders the full lead detail view with score breakdown.
 */
export default async function LeadDetailPage({
  params,
}: LeadDetailPageProps): Promise<React.ReactNode> {
  const { org: orgSlug, space: spaceSlug, board: boardSlug, thread: threadSlug } = await params;

  const [thread, { data: messages }] = await Promise.all([
    fetchThread(orgSlug, spaceSlug, boardSlug, threadSlug),
    fetchMessages(orgSlug, spaceSlug, boardSlug, threadSlug),
  ]);

  // Parse optional score breakdown and lead data from metadata.
  let metaObj: unknown = null;
  try {
    metaObj = JSON.parse(thread.metadata);
  } catch {
    // metadata may not be valid JSON
  }
  const scoreBreakdown =
    metaObj && typeof metaObj === "object" && !Array.isArray(metaObj)
      ? parseScoreBreakdown((metaObj as Record<string, unknown>).score_breakdown ?? null)
      : null;

  const lead = parseLeadData(thread.metadata);
  const customerOrgHref = lead.customer_org_id ? `/orgs/${lead.customer_org_id}` : undefined;

  return (
    <div className="mx-auto max-w-5xl space-y-6 p-6">
      {/* Breadcrumbs */}
      <nav className="flex items-center gap-1 text-sm text-muted-foreground">
        <Link href="/crm" className="hover:text-foreground">
          CRM Pipeline
        </Link>
        <ChevronRight className="h-3.5 w-3.5" />
        <span className="font-medium text-foreground">{thread.title}</span>
      </nav>

      <LeadDetailView
        thread={thread}
        messages={messages}
        scoreBreakdown={scoreBreakdown}
        customerOrgHref={customerOrgHref}
      />
    </div>
  );
}
