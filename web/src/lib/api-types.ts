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
  messages?: Message[];
}

/** Message type enum matching Go MessageType. */
export type MessageType = "note" | "email" | "call_log" | "comment" | "system";

/** Message — single message within a Thread. */
export interface Message extends BaseEntity {
  thread_id: string;
  body: string;
  author_id: string;
  metadata: string;
  type: MessageType;
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

// --- Admin types (matching Go backend admin package) ---

/** Count stats with total and recent counts. */
export interface CountStats {
  total: number;
  last_7d: number;
  last_30d: number;
}

/** Platform-wide statistics from GET /v1/admin/stats. */
export interface PlatformStats {
  orgs: CountStats;
  users: CountStats;
  threads: CountStats;
  messages: CountStats;
  db_size_bytes: number;
  api_uptime_pct: number;
  failed_webhooks_24h: number;
  pending_notifications: number;
}

/** User shadow — local cache of Clerk user data. */
export interface UserShadow {
  clerk_user_id: string;
  email: string;
  display_name: string;
  avatar_url?: string;
  last_seen_at: string;
  is_banned: boolean;
  ban_reason?: string;
  synced_at: string;
  banned_at?: string | null;
  banned_by?: string;
}

/** Platform admin record. */
export interface PlatformAdmin {
  user_id: string;
  granted_by: string;
  granted_at: string;
  is_active: boolean;
}

/** Feature flag toggle. */
export interface FeatureFlag {
  key: string;
  enabled: boolean;
  org_scope?: string | null;
  updated_at: string;
}

/** Response from POST /v1/admin/users/{user_id}/impersonate. */
export interface ImpersonationResponse {
  token: string;
  expires_at: string;
}

/** Security log entry from GET /v1/admin/security/recent-logins or failed-auths. */
export interface SecurityLogEntry {
  id: string;
  user_id: string;
  ip_address: string;
  user_agent: string;
  timestamp: string;
}

// --- Admin usage types (matching Go backend admin package) ---

/** Time window for API usage queries. */
export type ApiUsagePeriod = "24h" | "7d" | "30d";

/** Single endpoint usage entry from GET /v1/admin/api-usage. */
export interface ApiUsageEntry {
  endpoint: string;
  method: string;
  count: number;
}

/** API usage response envelope. */
export interface ApiUsageResponse {
  period: string;
  data: ApiUsageEntry[];
}

/** Single LLM usage log entry from GET /v1/admin/llm-usage. */
export interface LlmUsageEntry {
  id: string;
  endpoint: string;
  model: string;
  input_tokens: number;
  output_tokens: number;
  duration_ms: number;
  created_at: string;
}

/** LLM usage response envelope. */
export interface LlmUsageResponse {
  data: LlmUsageEntry[];
  message: string;
}

/** Export type options for admin data exports. */
export type AdminExportType = "users" | "orgs" | "audit";

/** Export format options. */
export type AdminExportFormat = "csv" | "json";

/** Export status values. */
export type AdminExportStatus = "pending" | "processing" | "completed" | "failed";

/** Admin data export record matching Go AdminExport model. */
export interface AdminExport {
  id: string;
  type: AdminExportType;
  filters: string;
  format: AdminExportFormat;
  status: AdminExportStatus;
  file_path?: string;
  requested_by: string;
  error_msg?: string;
  created_at: string;
  completed_at?: string | null;
}

// --- IO Channel types (matching Go backend models/channel-config.go) ---

// --- RBAC Policy types (matching Go backend admin/rbac-override.go) ---

/** Resolution strategy configuration. */
export interface RBACResolution {
  strategy: string;
  order: string[];
}

/** Role hierarchy and permissions. */
export interface RBACEffectiveRoles {
  hierarchy: string[];
  permissions: Record<string, string[]>;
}

/** Default role assignments per entity level. */
export interface RBACDefaults {
  org_member_role: string;
  space_member_role: string;
  board_member_role: string;
}

/** Effective RBAC policy (base + overrides) from GET /v1/admin/rbac-policy. */
export interface EffectivePolicy {
  resolution: RBACResolution;
  roles: RBACEffectiveRoles;
  defaults: RBACDefaults;
  overrides?: Record<string, unknown>;
}

/** Request body for POST /v1/admin/rbac-policy/preview. */
export interface RBACPreviewRequest {
  user_id: string;
  entity_type: string;
  entity_id: string;
}

/** Response from POST /v1/admin/rbac-policy/preview. */
export interface RBACPreviewResponse {
  user_id: string;
  entity_type: string;
  entity_id: string;
  role: string;
  permissions: string[];
}

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
