import { fetchBillingInfo } from "@/lib/admin-api";
import { BillingDashboard } from "@/components/admin/billing-dashboard";

export default async function AdminBillingPage(): Promise<React.ReactNode> {
  const billing = await fetchBillingInfo();

  return <BillingDashboard billing={billing} />;
}
