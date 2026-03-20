import Link from "next/link";
import { notFound } from "next/navigation";
import { revalidatePath } from "next/cache";
import { Phone } from "lucide-react";
import {
  fetchChannelConfig,
  fetchChannelHealth,
  fetchFirstOrgId,
  putChannelConfig,
} from "@/lib/admin-api";
import { ChannelConfigForm } from "@/components/admin/channel-config-form";
import { ChannelHealthBadge } from "@/components/admin/channel-health-badge";
import { ChatChannelPanel } from "@/components/admin/chat-channel-panel";
import { ChannelDetailDLQ } from "./channel-detail-dlq";
import type { ChannelType } from "@/lib/api-types";

const VALID_TYPES = new Set<string>(["email", "voice", "chat"]);

const CHANNEL_LABELS: Record<ChannelType, string> = {
  email: "Email",
  voice: "Voice",
  chat: "Chat",
};

interface PageProps {
  params: Promise<{ type: string }>;
}

export default async function ChannelDetailPage({ params }: PageProps): Promise<React.ReactNode> {
  const { type } = await params;

  if (!VALID_TYPES.has(type)) {
    notFound();
  }

  const channelType = type as ChannelType;
  const orgId = await fetchFirstOrgId();

  const [config, health] = await Promise.allSettled([
    fetchChannelConfig(orgId, channelType),
    fetchChannelHealth(orgId, channelType),
  ]);

  const configData = config.status === "fulfilled" ? config.value : null;
  const healthData = health.status === "fulfilled" ? health.value : null;

  return (
    <div data-testid="channel-detail-page" className="flex flex-col gap-6">
      {/* Breadcrumb */}
      <nav className="text-sm text-muted-foreground" data-testid="channel-breadcrumb">
        <Link href="/admin" className="hover:text-foreground">
          Admin
        </Link>
        <span className="mx-1">→</span>
        <Link href="/admin/channels" className="hover:text-foreground">
          Channels
        </Link>
        <span className="mx-1">→</span>
        <span className="text-foreground font-medium">{CHANNEL_LABELS[channelType]}</span>
      </nav>

      {/* Health status */}
      {healthData && (
        <div className="flex items-center gap-3" data-testid="channel-detail-health">
          <span className="text-sm text-muted-foreground">Health:</span>
          <ChannelHealthBadge status={healthData.status} />
        </div>
      )}

      {/* Manage Numbers link for voice channel */}
      {channelType === "voice" && (
        <Link
          href="/admin/channels/voice/numbers"
          data-testid="manage-numbers-link"
          className="inline-flex items-center gap-1.5 rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-accent"
        >
          <Phone className="h-4 w-4" />
          Manage Numbers
        </Link>
      )}

      {/* Layout: side-by-side for chat, stacked for others */}
      {channelType === "chat" ? (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2" data-testid="chat-layout">
          <div className="flex flex-col gap-6">
            {/* Config form */}
            <ChannelConfigForm
              channelType={channelType}
              initialConfig={configData}
              onSave={async (settings, enabled) => {
                "use server";
                await putChannelConfig(orgId, channelType, {
                  settings: JSON.stringify(settings),
                  enabled,
                });
                revalidatePath(`/admin/channels/${channelType}`);
              }}
            />
          </div>

          {/* Right-side preview panel */}
          <ChatChannelPanel embedKey={configData?.org_id ?? orgId} />
        </div>
      ) : (
        <ChannelConfigForm
          channelType={channelType}
          initialConfig={configData}
          onSave={async (settings, enabled) => {
            "use server";
            await putChannelConfig(orgId, channelType, {
              settings: JSON.stringify(settings),
              enabled,
            });
            revalidatePath(`/admin/channels/${channelType}`);
          }}
        />
      )}

      {/* DLQ monitor (client component) */}
      <ChannelDetailDLQ org={orgId} channelType={channelType} />
    </div>
  );
}
