// API types matching Go backend models in api/internal/models/.

/** Base fields present on every entity. */
export interface BaseEntity {
  id: string;
  created_at: string;
  updated_at: string;
  deleted_at?: string | null;
}

/** Organization — top-level entity. */
export interface Org extends BaseEntity {
  name: string;
  slug: string;
  description?: string;
  metadata: string;
  billing_tier?: string;
  payment_status?: string;
  spaces?: Space[];
}

/** Space type enum matching Go SpaceType. */
export type SpaceType = "general" | "crm" | "support" | "community" | "knowledge_base";

/** Space — categorized area within an Org. */
export interface Space extends BaseEntity {
  org_id: string;
  name: string;
  slug: string;
  description?: string;
  metadata: string;
  type: SpaceType;
  boards?: Board[];
}

/** Board — topic board within a Space. */
export interface Board extends BaseEntity {
  space_id: string;
  name: string;
  slug: string;
  description?: string;
  metadata: string;
  is_locked: boolean;
  threads?: Thread[];
}

/**
 * ThreadWithAuthor extends Thread with resolved creator and org display info.
 * These optional fields are populated by the globalspace endpoints when the
 * author or org can be resolved from local shadow/org tables.
 */
export interface ThreadWithAuthor extends Thread {
  /** Email address of the thread author, if available. */
  author_email?: string;
  /** Display name of the thread author, if available. */
  author_name?: string;
  /** Name of the org associated with this thread, if available. */
  org_name?: string;
}

/** Thread — discussion thread within a Board. */
export interface Thread extends BaseEntity {
  board_id: string;
  title: string;
  body?: string;
  slug: string;
  metadata: string;
  author_id: string;
  is_pinned: boolean;
  is_locked: boolean;
  is_hidden: boolean;
  vote_score: number;
  status?: string;
  priority?: string;
  stage?: string;
  assigned_to?: string;
  /** Sequential human-readable ticket number; only set for support threads. */
  ticket_number?: number;
  messages?: Message[];
}

/** Message type enum matching Go MessageType. */
export type MessageType = "note" | "email" | "call_log" | "comment" | "system";

/**
 * Support-specific entry type enum.
 * customer     — Message from the ticket creator (manual or from inbound email).
 * agent_reply  — Published DEFT agent reply visible to the customer.
 * draft        — Unpublished agent reply, invisible to the customer.
 * context      — DEFT-internal note, never shown to the customer.
 * system_event — System-inserted event (ticket created, closed, reopened, etc.).
 */
export type SupportEntryType = "customer" | "agent_reply" | "draft" | "context" | "system_event";

/** Union of all valid message type values. */
export type AnyMessageType = MessageType | SupportEntryType;

/** Message — single message within a Thread. */
export interface Message extends BaseEntity {
  thread_id: string;
  body: string;
  author_id: string;
  metadata: string;
  type: AnyMessageType;
}

/**
 * SupportEntry extends Message with support-specific lifecycle fields.
 * Returned by GET /v1/support/tickets/{slug}/entries.
 */
export interface SupportEntry extends BaseEntity {
  thread_id: string;
  body: string;
  author_id: string;
  metadata: string;
  type: SupportEntryType;
  /** When true the entry is hidden from any caller outside the DEFT org. */
  is_deft_only: boolean;
  /** Whether the entry has been published to the customer. */
  is_published: boolean;
  /** Whether the entry is locked against further edits. */
  is_immutable: boolean;
  /** ISO timestamp when a draft was promoted to agent_reply. */
  published_at?: string | null;
}

/** RBAC role enum matching Go Role. */
export type Role = "viewer" | "commenter" | "contributor" | "moderator" | "admin" | "owner";

/** Membership shared fields. */
interface BaseMembership extends BaseEntity {
  user_id: string;
  role: Role;
}

/** Org membership. */
export interface OrgMembership extends BaseMembership {
  org_id: string;
}

/** Space membership. */
export interface SpaceMembership extends BaseMembership {
  space_id: string;
}

/** Board membership. */
export interface BoardMembership extends BaseMembership {
  board_id: string;
}

/** In-app notification. */
export interface Notification extends BaseEntity {
  user_id: string;
  type: string;
  title: string;
  body?: string;
  entity_type?: string;
  entity_id?: string;
  is_read: boolean;
}

/** Cursor-based pagination metadata. */
export interface PageInfo {
  next_cursor?: string;
  has_more: boolean;
}

/** Paginated list response shape. */
export interface PaginatedResponse<T> {
  data: T[];
  page_info: PageInfo;
}

/** RFC 7807 Problem Details error response. */
export interface ProblemDetail {
  type: string;
  title: string;
  status: number;
  detail?: string;
  instance?: string;
}

/** File upload record. */
export interface Upload extends BaseEntity {
  org_id: string;
  entity_type: string;
  entity_id: string;
  filename: string;
  content_type: string;
  size: number;
  storage_path: string;
  uploader_id: string;
}

/** Search result item. */
export interface SearchResult {
  entity_type: string;
  entity_id: string;
  title: string;
  snippet: string;
  score: number;
}

/** Vote on a thread. */
export interface Vote extends BaseEntity {
  thread_id: string;
  user_id: string;
  weight: number;
}

/** Content flag for moderation. */
export interface Flag extends BaseEntity {
  thread_id: string;
  reporter_id: string;
  reason: string;
  status: "pending" | "resolved" | "dismissed";
  resolved_by?: string;
  resolution_note?: string;
}

/** Webhook subscription. */
export interface WebhookSubscription extends BaseEntity {
  org_id: string;
  scope_type: string;
  scope_id: string;
  url: string;
  event_filter: string;
  is_active: boolean;
}

/** Webhook delivery record. */
export interface WebhookDelivery extends BaseEntity {
  subscription_id: string;
  event_type: string;
  payload: string;
  status_code: number;
  attempts: number;
  next_retry_at?: string;
  completed_at?: string;
}

/** Audit log entry. */
export interface AuditEntry extends BaseEntity {
  user_id: string;
  action: string;
  entity_type: string;
  entity_id: string;
  before_state?: string;
  after_state?: string;
  ip_address?: string;
  request_id?: string;
}

/** Billing information for an org. */
export interface BillingInfo {
  org_id: string;
  tier: string;
  payment_status: string;
  invoices: Invoice[];
}

/** Invoice record. */
export interface Invoice {
  id: string;
  amount: number;
  currency: string;
  status: string;
  issued_at: string;
  due_at: string;
  paid_at?: string;
}

/** Revision history entry for threads/messages. */
export interface Revision extends BaseEntity {
  entity_type: string;
  entity_id: string;
  version: number;
  previous_content: string;
  editor_id: string;
}

/** Whether the current user has voted on a thread. */
export interface UserVoteStatus {
  voted: boolean;
}

/** Personal API key. */
export interface ApiKey {
  id: string;
  name: string;
  prefix: string;
  created_at: string;
  last_used_at?: string | null;
}

/** Response from creating a new API key (includes full key shown once). */
export interface ApiKeyCreateResponse {
  id: string;
  name: string;
  prefix: string;
  key: string;
  created_at: string;
}

/** Sort option for community thread lists. */
export type ThreadSortOption = "votes" | "newest" | "oldest";

/** Board option for moving threads. */
export interface BoardOption {
  id: string;
  name: string;
  slug: string;
}

/** Digest frequency enum. */
export type DigestFrequency = "none" | "daily" | "weekly";

/** Notification type enum for preference channels. */
export type NotificationType = "message" | "mention" | "stage_change" | "assignment";

/** Notification channel enum. */
export type NotificationChannel = "in_app" | "email";

/** Per-type, per-channel notification preference. */
export interface NotificationPreference extends BaseEntity {
  user_id: string;
  notification_type: NotificationType;
  channel: NotificationChannel;
  enabled: boolean;
}

/** Digest schedule preference. */
export interface DigestSchedule extends BaseEntity {
  user_id: string;
  frequency: DigestFrequency;
}

/** WebSocket event types emitted by the server. */
export type WSEventType =
  | "message.created"
  | "message.updated"
  | "thread.updated"
  | "typing"
  | "notification";

/** Base WebSocket message envelope. */
export interface WSMessage<T = unknown> {
  type: WSEventType;
  channel: string;
  payload: T;
  timestamp: string;
}

/** Typing event payload. */
export interface TypingPayload {
  user_id: string;
  user_name?: string;
  thread_id: string;
}

/** Client-to-server subscription command. */
export interface WSSubscribeCommand {
  action: "subscribe" | "unsubscribe";
  channel: string;
}

/** Client-to-server typing command. */
export interface WSTypingCommand {
  action: "typing";
  thread_id: string;
}

/** Union of client-to-server WS commands. */
export type WSClientCommand = WSSubscribeCommand | WSTypingCommand;

// Re-export domain-specific type modules for backwards compatibility.
export * from "./api-types-admin";
export * from "./api-types-channel";
