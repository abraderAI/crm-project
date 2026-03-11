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

/** Entity revision (thread or message edit history). */
export interface Revision extends BaseEntity {
  entity_type: string;
  entity_id: string;
  version: number;
  previous_content: string;
  editor_id: string;
}

/** Search result item. */
export interface SearchResult {
  entity_type: string;
  entity_id: string;
  title: string;
  snippet: string;
  score: number;
}
