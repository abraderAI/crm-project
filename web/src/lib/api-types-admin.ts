// Admin types matching Go backend admin package.

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

// --- Admin usage types ---

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

/** Admin org detail — org with aggregate counts from GET /v1/admin/orgs/{org}. */
export interface AdminOrgDetail {
  id: string;
  name: string;
  slug: string;
  description?: string;
  metadata: string;
  billing_tier?: string;
  payment_status?: string;
  suspended_at?: string | null;
  suspend_reason?: string;
  created_at: string;
  updated_at: string;
  /** Number of members in this org. */
  member_count: number;
  /** Number of spaces in this org. */
  space_count: number;
  /** Number of boards across all spaces. */
  board_count: number;
  /** Number of threads across all boards. */
  thread_count: number;
}
