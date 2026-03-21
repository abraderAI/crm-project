import type { ResolvedUser } from "@/lib/use-user-directory";

export interface UserLabelProps {
  /** The raw user ID. */
  userId: string;
  /** Resolved user info (from useUserDirectory.resolve). */
  resolved?: ResolvedUser;
  /** Show the org badge. Defaults to true. */
  showOrg?: boolean;
}

/** Displays a user's name with an optional org badge. Falls back to truncated ID. */
export function UserLabel({ userId, resolved, showOrg = true }: UserLabelProps): React.ReactNode {
  const name = resolved?.display_name ?? (userId.length > 16 ? `${userId.slice(0, 16)}…` : userId);

  return (
    <span className="inline-flex items-center gap-1.5" data-testid={`user-label-${userId}`}>
      <span className="text-foreground">{name}</span>
      {showOrg && resolved?.org_name && (
        <span
          className="rounded-full bg-blue-50 px-1.5 py-0.5 text-[10px] font-medium text-blue-700"
          data-testid={`user-label-org-${userId}`}
        >
          {resolved.org_name}
        </span>
      )}
    </span>
  );
}
