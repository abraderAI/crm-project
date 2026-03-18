import { LeadsManagementView } from "@/components/crm/leads-management-view";

/**
 * Leads management page for DEFT sales staff.
 * Access is controlled client-side via tier; tier 6 and 5 see all leads,
 * tier 4 sales reps see only their own and assigned leads.
 */
export default function LeadsPage(): React.ReactNode {
  return (
    <div className="mx-auto max-w-5xl p-6">
      <LeadsManagementView />
    </div>
  );
}
