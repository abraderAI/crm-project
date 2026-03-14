import { fetchFlags } from "@/lib/admin-api";
import { ModerationView } from "@/components/community/moderation-view";

export default async function AdminModerationPage(): Promise<React.ReactNode> {
  const flags = await fetchFlags();

  return <ModerationView initialFlags={flags.data} />;
}
