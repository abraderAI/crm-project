// IO Channel types matching Go backend models/channel-config.go.

/** Channel type enum matching backend channel types. */
export type ChannelType = "email" | "voice" | "chat";

/** Dead-letter queue event status. */
export type DLQStatus = "failed" | "retrying" | "resolved" | "dismissed";

/** Channel configuration for a specific channel type. */
export interface ChannelConfig {
  id: string;
  org_id: string;
  channel_type: ChannelType;
  settings: string;
  enabled: boolean;
}

/** Health status for a channel. */
export interface ChannelHealth {
  channel_type: ChannelType;
  enabled: boolean;
  last_event_at: string;
  error_rate: number;
  status: string;
}

/** Dead-letter queue event. */
export interface DeadLetterEvent {
  id: string;
  org_id: string;
  channel_type: ChannelType;
  event_payload: string;
  error_message: string;
  attempts: number;
  last_attempt_at: string;
  status: DLQStatus;
  created_at: string;
}

/** Owned phone number from LiveKit. */
export interface PhoneNumber {
  phone_number: string;
  status: string;
  dispatch_rule_id: string;
  purchased_at: string;
}

/** Available phone number from search results. */
export interface PhoneNumberSearchResult {
  phone_number: string;
  country: string;
  area_code: string;
  monthly_cost: string;
}
