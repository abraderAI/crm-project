import type { PaginatedResponse, ProblemDetail } from "./api-types";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

/** Error thrown when the API returns a non-OK response. */
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly problem: ProblemDetail,
  ) {
    super(problem.detail ?? problem.title);
    this.name = "ApiError";
  }
}

/** Options for API requests. */
export interface ApiRequestOptions {
  token?: string | null;
  headers?: Record<string, string>;
  signal?: AbortSignal;
  cache?: RequestCache;
  next?: NextFetchRequestConfig;
}

/** Build headers for an API request, including auth if token present. */
export function buildHeaders(
  token?: string | null,
  extra?: Record<string, string>,
): Record<string, string> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    Accept: "application/json",
    ...extra,
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }
  return headers;
}

/** Construct a full API URL from a path and optional query params. */
export function buildUrl(path: string, params?: Record<string, string>): string {
  const url = new URL(`/v1${path}`, API_BASE_URL);
  if (params) {
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined && value !== "") {
        url.searchParams.set(key, value);
      }
    }
  }
  return url.toString();
}

/** Parse an API response, throwing ApiError on non-OK status. */
export async function parseResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    let problem: ProblemDetail;
    try {
      problem = (await response.json()) as ProblemDetail;
    } catch {
      problem = {
        type: "about:blank",
        title: response.statusText || "Request failed",
        status: response.status,
        detail: `HTTP ${response.status}`,
      };
    }
    throw new ApiError(response.status, problem);
  }
  return (await response.json()) as T;
}

/**
 * Server-side API fetch for RSC.
 * Use in server components: `const orgs = await serverFetch<Org[]>("/orgs", { token });`
 */
export async function serverFetch<T>(path: string, options?: ApiRequestOptions): Promise<T> {
  const url = buildUrl(path);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(options?.token, options?.headers),
    signal: options?.signal,
    cache: options?.cache ?? "no-store",
    next: options?.next,
  });
  return parseResponse<T>(response);
}

/**
 * Server-side paginated fetch for RSC.
 */
export async function serverFetchPaginated<T>(
  path: string,
  params?: Record<string, string>,
  options?: ApiRequestOptions,
): Promise<PaginatedResponse<T>> {
  const url = buildUrl(path, params);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(options?.token, options?.headers),
    signal: options?.signal,
    cache: options?.cache ?? "no-store",
    next: options?.next,
  });
  return parseResponse<PaginatedResponse<T>>(response);
}

/**
 * Client-side mutation helper for use in Client Components.
 * Supports POST, PATCH, PUT, DELETE.
 */
export async function clientMutate<T>(
  method: "POST" | "PATCH" | "PUT" | "DELETE",
  path: string,
  options?: ApiRequestOptions & { body?: unknown },
): Promise<T> {
  const url = buildUrl(path);
  const response = await fetch(url, {
    method,
    headers: buildHeaders(options?.token, options?.headers),
    body: options?.body !== undefined ? JSON.stringify(options.body) : undefined,
    signal: options?.signal,
  });
  return parseResponse<T>(response);
}
