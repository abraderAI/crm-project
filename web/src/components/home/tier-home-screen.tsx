"use client";

import type { ReactNode } from "react";
import { useTier } from "@/hooks/use-tier";
import { useHomeLayout } from "@/hooks/use-home-layout";
import type { ProfileData } from "./widgets/my-profile-widget";
import { Tier1Home } from "./tier-1-home";
import { Tier2Home } from "./tier-2-home";
import { Tier3Home } from "./tier-3-home";
import { Tier4HomeScreen } from "./tier4-home-screen";
import { Tier5Home } from "./tier-5-home";
import { Tier6HomeScreen } from "./tier6-home-screen";

interface TierHomeScreenProps {
  /** Clerk auth token (null for anonymous users). */
  token: string | null;
  /** User profile data (for Tier 2 profile widget). */
  profile?: ProfileData | null;
  /** Whether the profile is loading. */
  profileLoading?: boolean;
}

/**
 * Top-level home screen component that renders the appropriate
 * tier-specific home screen based on the user's resolved tier.
 * Tiers 4 and 6 render a placeholder pending future phases.
 */
export function TierHomeScreen({ token, profile, profileLoading }: TierHomeScreenProps): ReactNode {
  const { tier, subType, orgId, isLoading: tierLoading, deftDepartment } = useTier();
  const { layout, isLoading: layoutLoading } = useHomeLayout(tier, token, deftDepartment);

  if (tierLoading || layoutLoading) {
    return (
      <div data-testid="tier-home-loading" className="mx-auto max-w-5xl p-6">
        <div className="animate-pulse space-y-4">
          <div className="h-6 w-48 rounded bg-muted" />
          <div className="grid gap-4 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="h-32 rounded-lg bg-muted" />
            ))}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div data-testid="tier-home-screen" className="mx-auto max-w-5xl p-6">
      {tier === 1 && <Tier1Home layout={layout} />}

      {tier === 2 && (
        <Tier2Home
          layout={layout}
          token={token ?? ""}
          profile={profile ?? null}
          profileLoading={profileLoading}
        />
      )}

      {tier === 3 && (
        <Tier3Home layout={layout} token={token ?? ""} orgId={orgId ?? ""} subType={subType} />
      )}

      {tier === 4 && (
        <Tier4HomeScreen
          token={token ?? ""}
          department={deftDepartment ?? "sales"}
          layout={layout}
        />
      )}

      {tier === 5 && <Tier5Home layout={layout} token={token ?? ""} orgId={orgId ?? ""} />}

      {tier === 6 && <Tier6HomeScreen token={token ?? ""} layout={layout} />}
    </div>
  );
}
