import { OpportunityForm } from "@/components/crm/opportunity-form";

interface OpportunityDetailPageProps {
  params: Promise<{ id: string }>;
}

/** Opportunity detail page under /crm/pipeline/[id]. Renders edit form for now (detail view extends lead-detail). */
export default async function OpportunityDetailPage({ params }: OpportunityDetailPageProps): Promise<React.ReactNode> {
  const { id } = await params;
  return (
    <div className="mx-auto max-w-5xl p-6">
      <OpportunityForm opportunityId={id} />
    </div>
  );
}
