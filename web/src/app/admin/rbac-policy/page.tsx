import { fetchRBACPolicy } from "@/lib/admin-api";
import { RBACPolicyEditor } from "@/components/admin/rbac-policy-editor";
import { RBACPolicyPreview } from "@/components/admin/rbac-policy-preview";

export default async function RBACPolicyPage(): Promise<React.ReactNode> {
  const policy = await fetchRBACPolicy();

  return (
    <div data-testid="rbac-policy-page" className="flex flex-col gap-8">
      <RBACPolicyEditor policy={policy} />
      <RBACPolicyPreview />
    </div>
  );
}
