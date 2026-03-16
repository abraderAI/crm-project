"use client";

import Link from "next/link";
import { useAuth, UserProfile } from "@clerk/nextjs";
import { Bell, ChevronRight, Key, Shield, User } from "lucide-react";
import { useEffect, useState } from "react";

import { useTier } from "@/hooks/use-tier";
import { TIER_LABELS } from "@/lib/tier-types";
import { ApiKeys } from "./api-keys";

/** User profile and account settings view. */
export function SettingsView(): React.ReactNode {
  const { getToken } = useAuth();
  const { tier, isLoading: tierLoading } = useTier();
  const [token, setToken] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    getToken().then((t) => {
      if (active) setToken(t);
    });
    return () => {
      active = false;
    };
  }, [getToken]);

  return (
    <div className="mx-auto max-w-3xl space-y-8 p-6" data-testid="settings-page">
      <h1 className="text-2xl font-bold text-foreground">Account Settings</h1>

      {/* Profile section — Clerk UserProfile */}
      <section data-testid="settings-profile-section" className="space-y-3">
        <div className="flex items-center gap-2">
          <User className="h-5 w-5 text-muted-foreground" />
          <h2 className="text-lg font-semibold text-foreground">Profile</h2>
        </div>
        <div className="rounded-lg border border-foreground/10 p-4">
          <UserProfile
            routing="hash"
            appearance={{
              elements: {
                rootBox: "w-full",
                cardBox: "shadow-none w-full",
              },
            }}
          />
        </div>
      </section>

      {/* Notifications section — link card */}
      <section data-testid="settings-notifications-section" className="space-y-3">
        <div className="flex items-center gap-2">
          <Bell className="h-5 w-5 text-muted-foreground" />
          <h2 className="text-lg font-semibold text-foreground">Notifications</h2>
        </div>
        <Link
          href="/notifications/preferences"
          data-testid="notifications-preferences-link"
          className="flex items-center justify-between rounded-lg border border-foreground/10 p-4 transition-colors hover:bg-foreground/5"
        >
          <div>
            <p className="text-sm font-medium text-foreground">Notification Preferences</p>
            <p className="text-sm text-muted-foreground">
              Manage email and in-app notification settings
            </p>
          </div>
          <ChevronRight className="h-5 w-5 text-muted-foreground" />
        </Link>
      </section>

      {/* API Keys section */}
      <section data-testid="settings-api-keys-section" className="space-y-3">
        <div className="flex items-center gap-2">
          <Key className="h-5 w-5 text-muted-foreground" />
        </div>
        {token ? (
          <ApiKeys token={token} />
        ) : (
          <div className="py-4 text-center text-sm text-muted-foreground">Loading API keys...</div>
        )}
      </section>

      {/* Current Tier section */}
      <section data-testid="settings-tier-section" className="space-y-3">
        <div className="flex items-center gap-2">
          <Shield className="h-5 w-5 text-muted-foreground" />
          <h2 className="text-lg font-semibold text-foreground">Current Tier</h2>
        </div>
        <div className="rounded-lg border border-foreground/10 p-4">
          {tierLoading ? (
            <p className="text-sm text-muted-foreground">Loading tier information...</p>
          ) : (
            <div className="flex items-center gap-3">
              <span
                data-testid="tier-badge"
                className="inline-flex items-center rounded-full bg-foreground/10 px-3 py-1 text-sm font-medium text-foreground"
              >
                Tier {tier}
              </span>
              <span data-testid="tier-label" className="text-sm text-muted-foreground">
                {TIER_LABELS[tier]}
              </span>
            </div>
          )}
        </div>
      </section>
    </div>
  );
}
