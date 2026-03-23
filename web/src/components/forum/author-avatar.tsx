import type { ReactNode } from "react";

/** Deterministic color from a string (author ID or name). */
function avatarColor(seed: string): string {
  const colors = [
    "bg-blue-500",
    "bg-emerald-500",
    "bg-violet-500",
    "bg-amber-500",
    "bg-rose-500",
    "bg-cyan-500",
    "bg-fuchsia-500",
    "bg-lime-500",
  ];
  let hash = 0;
  for (let i = 0; i < seed.length; i++) {
    hash = (hash * 31 + seed.charCodeAt(i)) | 0;
  }
  return colors[Math.abs(hash) % colors.length] ?? "bg-blue-500";
}

/** Extract initials from a display name or fall back to "?" */
function initials(name?: string): string {
  if (!name || name === "system-seed") return "?";
  const parts = name.trim().split(/\s+/);
  const first = parts[0]?.[0] ?? "";
  const second = parts[1]?.[0] ?? "";
  if (parts.length >= 2) return (first + second).toUpperCase();
  return name.slice(0, 2).toUpperCase();
}

interface AuthorAvatarProps {
  authorId: string;
  authorName?: string;
  size?: "sm" | "md";
}

/** Colored circle with author initials. */
export function AuthorAvatar({ authorId, authorName, size = "sm" }: AuthorAvatarProps): ReactNode {
  const sizeClass = size === "md" ? "h-10 w-10 text-sm" : "h-7 w-7 text-xs";
  return (
    <div
      data-testid="author-avatar"
      className={`${avatarColor(authorId)} ${sizeClass} flex shrink-0 items-center justify-center rounded-full font-semibold text-white`}
    >
      {initials(authorName)}
    </div>
  );
}
