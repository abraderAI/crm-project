import { OpportunityForm } from "@/components/crm/opportunity-form";

/** Create new opportunity page. */
export default function NewOpportunityPage(): React.ReactNode {
  return (
    <div className="mx-auto max-w-5xl p-6">
      <OpportunityForm />
    </div>
  );
}
