import { fetchWebhookSubscriptions, fetchWebhookDeliveries } from "@/lib/admin-api";
import { WebhookView } from "@/components/admin/webhook-view";

export default async function AdminWebhooksPage(): Promise<React.ReactNode> {
  const [subscriptions, deliveries] = await Promise.all([
    fetchWebhookSubscriptions(),
    fetchWebhookDeliveries(),
  ]);

  return (
    <WebhookView initialSubscriptions={subscriptions.data} initialDeliveries={deliveries.data} />
  );
}
