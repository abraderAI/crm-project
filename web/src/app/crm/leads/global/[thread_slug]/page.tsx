import Link from "next/link";
import { ChevronRight } from "lucide-react";

import { fetchGlobalLeadThread } from "@/lib/user-api";
import { parseScoreBreakdown, parseLeadData } from "@/lib/crm-types";
import type { Message } from "@/lib/api-types";
import { LeadDetailView } from "@/components/crm/lead-detail-view";

interface GlobalLeadDetailPageProps {
  params: Promise<{ thread_slug: string }>;
}

/**
 * Global lead detail page — fetches a single lead from global-leads space
 * and renders the full lead detail view with score breakdown.
 * Note: Global-space message history is not yet available via a server-side
 * endpoint; messages are passed as an empty array until that API is added.
 */
export default async function GlobalLeadDetailPage({
  params,
}: GlobalLeadDetailPageProps): Promise<React.ReactNode> {
  const { thread_slug: threadSlug } = await params;

  const thread = await fetchGlobalLeadThread(threadSlug);
  // Global space messages endpoint is not yet implemented; use empty array.
  const messages: Message[] = [];

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
        <Link href="/crm/leads" className="hover:text-foreground">
          Leads
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
