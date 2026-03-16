"use client";

import { useCallback, useEffect, useState } from "react";
import { DLQMonitor } from "@/components/admin/dlq-monitor";
import { buildHeaders, buildUrl, parseResponse } from "@/lib/api-client";
import type { ChannelType, DeadLetterEvent, PaginatedResponse } from "@/lib/api-types";

export interface ChannelDetailDLQProps {
  org: string;
  channelType: ChannelType;
}

/** Client-side DLQ fetcher and wrapper for the DLQ monitor component. */
export function ChannelDetailDLQ({ org, channelType }: ChannelDetailDLQProps): React.ReactNode {
  const [events, setEvents] = useState<DeadLetterEvent[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchEvents = useCallback(async (): Promise<void> => {
    try {
      setLoading(true);
      // channel_type is passed as a query param; no per-type path segment in backend.
      const url = buildUrl(`/orgs/${org}/channels/dlq`, { channel_type: channelType });
      const response = await fetch(url, {
        method: "GET",
        headers: buildHeaders(),
      });
      const data = await parseResponse<PaginatedResponse<DeadLetterEvent>>(response);
      setEvents(data.data);
    } catch {
      // Silently handle fetch errors; events remain empty.
    } finally {
      setLoading(false);
    }
  }, [org, channelType]);

  useEffect(() => {
    void fetchEvents();
  }, [fetchEvents]);

  const handleRetry = useCallback(
    async (eventId: string): Promise<void> => {
      try {
        const url = buildUrl(`/orgs/${org}/channels/dlq/${eventId}/retry`);
        await fetch(url, { method: "POST", headers: buildHeaders() });
        await fetchEvents();
      } catch {
        // Silently handle retry errors.
      }
    },
    [org, fetchEvents],
  );

  const handleDismiss = useCallback(
    async (eventId: string): Promise<void> => {
      try {
        const url = buildUrl(`/orgs/${org}/channels/dlq/${eventId}/dismiss`);
        await fetch(url, { method: "POST", headers: buildHeaders() });
        await fetchEvents();
      } catch {
        // Silently handle dismiss errors.
      }
    },
    [org, fetchEvents],
  );

  return (
    <DLQMonitor
      org={org}
      channelType={channelType}
      events={events}
      loading={loading}
      onRetry={(id) => void handleRetry(id)}
      onDismiss={(id) => void handleDismiss(id)}
      onRefresh={() => void fetchEvents()}
    />
  );
}
