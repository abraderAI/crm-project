import type { Thread } from "@/lib/api-types";
import { fetchSupportTickets } from "@/lib/user-api";
import { SupportView } from "@/components/support/support-view";

/** Support tickets page — shows the current user's tickets with inline create. */
export default async function SupportPage(): Promise<React.ReactNode> {
  let tickets: Thread[] = [];
  try {
    const result = await fetchSupportTickets();
    tickets = result.data;
  } catch {
    // Render with empty list on error — SupportView handles the empty state.
  }

  return (
    <div className="mx-auto max-w-3xl p-6">
      <SupportView initialTickets={tickets} />
    </div>
  );
}
