import { fetchChannelHealth } from "@/lib/admin-api";
import { ChannelOverview } from "@/components/admin/channel-overview";
import type { ChannelHealth, ChannelType } from "@/lib/api-types";

const CHANNEL_TYPES: ChannelType[] = ["email", "voice", "chat"];
const DEFAULT_ORG = "default";

export default async function AdminChannelsPage(): Promise<React.ReactNode> {
  const results = await Promise.allSettled(
    CHANNEL_TYPES.map((type) => fetchChannelHealth(DEFAULT_ORG, type)),
  );

  const healthMap: Record<ChannelType, ChannelHealth | null> = {
    email: null,
    voice: null,
    chat: null,
  };

  for (let i = 0; i < CHANNEL_TYPES.length; i++) {
    const result = results[i];
    const channelType = CHANNEL_TYPES[i];
    if (result && channelType && result.status === "fulfilled") {
      healthMap[channelType] = result.value;
    }
  }

  return <ChannelOverview healthMap={healthMap} />;
}
