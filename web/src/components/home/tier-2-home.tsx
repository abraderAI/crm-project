"use client";

import type { ReactNode } from "react";
import { WIDGET_IDS } from "@/lib/default-layouts";
import type { WidgetConfig } from "@/lib/tier-types";
import { HomeLayout, type WidgetRegistry } from "./home-layout";
import { MyProfileWidget, type ProfileData } from "./widgets/my-profile-widget";
import { MyForumActivityWidget } from "./widgets/my-forum-activity-widget";
import { MySupportTicketsWidget } from "./widgets/my-support-tickets-widget";
import { UpgradeCTAWidget } from "./widgets/upgrade-cta-widget";

interface Tier2HomeProps {
  /** Widget layout config (from useHomeLayout or default). */
  layout: WidgetConfig[];
  /** Auth token for authenticated API calls. */
  token: string;
  /** User profile data for the profile widget. */
  profile: ProfileData | null;
  /** Whether the profile is loading. */
  profileLoading?: boolean;
}

/**
 * Tier 2 (Registered Developer) home screen.
 * Renders profile, forum activity, support tickets, and upgrade CTA.
 */
export function Tier2Home({ layout, token, profile, profileLoading }: Tier2HomeProps): ReactNode {
  /** Widget registry for Tier 2 home screen. */
  const registry: WidgetRegistry = {
    [WIDGET_IDS.MY_PROFILE]: {
      title: "My Profile",
      render: () => <MyProfileWidget profile={profile} isLoading={profileLoading} />,
    },
    [WIDGET_IDS.MY_FORUM_ACTIVITY]: {
      title: "Forum Activity",
      render: () => <MyForumActivityWidget token={token} />,
    },
    [WIDGET_IDS.MY_SUPPORT_TICKETS]: {
      title: "Support Tickets",
      render: () => <MySupportTicketsWidget token={token} />,
    },
    [WIDGET_IDS.UPGRADE_CTA]: {
      title: "Upgrade",
      render: () => <UpgradeCTAWidget />,
    },
  };

  return (
    <div data-testid="tier-2-home">
      <h2 className="mb-4 text-lg font-semibold text-foreground">Home</h2>
      <HomeLayout layout={layout} registry={registry} />
    </div>
  );
}
