import { SupportManagementView } from "@/components/support/support-management-view";

/**
 * Support tickets page — renders the tier-aware SupportManagementView.
 * RBAC and data fetching are handled client-side in the component.
 */
export default function SupportPage(): React.ReactNode {
  return (
    <div className="mx-auto max-w-3xl p-6">
      <SupportManagementView />
    </div>
  );
}
