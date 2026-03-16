"use client";

import type { ReactNode } from "react";
import Link from "next/link";
import Image from "next/image";
import { User, Mail, CheckCircle } from "lucide-react";

/** Profile data shape expected by the widget. */
export interface ProfileData {
  displayName: string;
  email: string;
  avatarUrl?: string | null;
  accountStatus?: string;
}

interface MyProfileWidgetProps {
  /** User profile information. */
  profile: ProfileData | null;
  /** Whether the profile is still loading. */
  isLoading?: boolean;
}

/** Displays the authenticated user's name, email, account status, and edit link. */
export function MyProfileWidget({ profile, isLoading }: MyProfileWidgetProps): ReactNode {
  if (isLoading) {
    return (
      <div data-testid="my-profile-loading" className="animate-pulse space-y-2">
        <div className="h-4 w-3/4 rounded bg-muted" />
        <div className="h-4 w-1/2 rounded bg-muted" />
      </div>
    );
  }

  if (!profile) {
    return (
      <p data-testid="my-profile-empty" className="text-sm text-muted-foreground">
        Profile information unavailable.
      </p>
    );
  }

  return (
    <div data-testid="my-profile-widget" className="space-y-3">
      <div className="flex items-center gap-3">
        {profile.avatarUrl ? (
          <Image
            src={profile.avatarUrl}
            alt={profile.displayName}
            width={40}
            height={40}
            className="h-10 w-10 rounded-full"
            data-testid="my-profile-avatar"
          />
        ) : (
          <div
            className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10"
            data-testid="my-profile-avatar-placeholder"
          >
            <User className="h-5 w-5 text-primary" />
          </div>
        )}
        <div>
          <p className="text-sm font-medium text-foreground" data-testid="my-profile-name">
            {profile.displayName}
          </p>
          <div className="flex items-center gap-1 text-xs text-muted-foreground">
            <Mail className="h-3 w-3" />
            <span data-testid="my-profile-email">{profile.email}</span>
          </div>
        </div>
      </div>

      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1 text-xs text-muted-foreground">
          <CheckCircle className="h-3 w-3 text-green-500" />
          <span data-testid="my-profile-status">{profile.accountStatus ?? "Active"}</span>
        </div>
        <Link
          href="/settings/profile"
          data-testid="my-profile-edit-link"
          className="text-xs font-medium text-primary hover:underline"
        >
          Edit profile
        </Link>
      </div>
    </div>
  );
}
