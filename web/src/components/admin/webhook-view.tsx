"use client";

import { useCallback, useState } from "react";
import { useAuth } from "@clerk/nextjs";

import type { WebhookSubscription, WebhookDelivery } from "@/lib/api-types";
import {
  createWebhook,
  deleteWebhook,
  toggleWebhook,
  replayWebhookDelivery,
} from "@/lib/entity-api";
import { WebhookManager } from "./webhook-manager";
import { WebhookDeliveryLog } from "./webhook-delivery-log";

export interface WebhookViewProps {
  /** Initial webhook subscriptions fetched on the server. */
  initialSubscriptions: WebhookSubscription[];
  /** Initial delivery log entries fetched on the server. */
  initialDeliveries: WebhookDelivery[];
}

/** Client wrapper wiring WebhookManager and WebhookDeliveryLog to entity-api mutations. */
export function WebhookView({
  initialSubscriptions,
  initialDeliveries,
}: WebhookViewProps): React.ReactNode {
  const { getToken } = useAuth();
  const [subscriptions, setSubscriptions] = useState(initialSubscriptions);
  const [deliveries] = useState(initialDeliveries);

  const handleCreate = useCallback(
    async (url: string, eventFilter: string) => {
      const token = await getToken();
      if (!token) return;
      const created = await createWebhook(token, url, eventFilter);
      setSubscriptions((prev) => [...prev, created]);
    },
    [getToken],
  );

  const handleDelete = useCallback(
    async (subscriptionId: string) => {
      const token = await getToken();
      if (!token) return;
      await deleteWebhook(token, subscriptionId);
      setSubscriptions((prev) => prev.filter((s) => s.id !== subscriptionId));
    },
    [getToken],
  );

  const handleToggle = useCallback(
    async (subscriptionId: string) => {
      const token = await getToken();
      if (!token) return;
      const updated = await toggleWebhook(token, subscriptionId);
      setSubscriptions((prev) => prev.map((s) => (s.id === subscriptionId ? updated : s)));
    },
    [getToken],
  );

  const handleReplay = useCallback(
    async (deliveryId: string) => {
      const token = await getToken();
      if (!token) return;
      await replayWebhookDelivery(token, deliveryId);
    },
    [getToken],
  );

  return (
    <div className="flex flex-col gap-8">
      <WebhookManager
        subscriptions={subscriptions}
        onCreate={handleCreate}
        onDelete={handleDelete}
        onToggle={handleToggle}
      />
      <WebhookDeliveryLog deliveries={deliveries} onReplay={handleReplay} />
    </div>
  );
}
