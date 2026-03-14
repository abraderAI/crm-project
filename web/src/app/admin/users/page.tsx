import { fetchAdminUsers } from "@/lib/admin-api";

/** Format a date string for display. */
function formatDate(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    return d.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return dateStr;
  }
}

export default async function AdminUsersPage(): Promise<React.ReactNode> {
  const { data: users } = await fetchAdminUsers();

  return (
    <div data-testid="admin-users" className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-foreground">Users ({users.length})</h2>
      </div>

      {users.length === 0 ? (
        <p className="py-8 text-center text-sm text-muted-foreground" data-testid="users-empty">
          No users found.
        </p>
      ) : (
        <div
          className="divide-y divide-border rounded-lg border border-border"
          data-testid="user-list"
        >
          {users.map((user) => (
            <div
              key={user.clerk_user_id}
              className="flex items-center gap-4 px-4 py-3"
              data-testid={`user-row-${user.clerk_user_id}`}
            >
              <div className="flex flex-col">
                <span className="text-sm font-medium text-foreground">
                  {user.display_name || user.email}
                </span>
                <span className="text-xs text-muted-foreground">{user.email}</span>
              </div>

              <span className="ml-auto text-xs text-muted-foreground">
                Last seen: {formatDate(user.last_seen_at)}
              </span>

              {user.is_banned && (
                <span
                  className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-800"
                  data-testid={`user-banned-${user.clerk_user_id}`}
                >
                  Banned
                </span>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
