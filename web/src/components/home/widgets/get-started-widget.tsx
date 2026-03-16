"use client";

import type { ReactNode } from "react";
import Link from "next/link";
import { Rocket } from "lucide-react";

/** Feature highlights shown in the CTA card. */
const FEATURES = [
  "Access community forums and documentation",
  "Create and track support tickets",
  "Connect with other developers",
] as const;

/** CTA card encouraging anonymous visitors to sign up. Visible to Tier 1 only. */
export function GetStartedWidget(): ReactNode {
  return (
    <div data-testid="get-started-widget" className="space-y-3">
      <div className="flex items-center gap-2">
        <Rocket className="h-5 w-5 text-primary" />
        <p className="text-sm font-medium text-foreground">Join the DEFT community today</p>
      </div>

      <ul className="space-y-1" data-testid="get-started-features">
        {FEATURES.map((feature) => (
          <li key={feature} className="flex items-start gap-2 text-sm text-muted-foreground">
            <span className="mt-0.5 text-primary">•</span>
            {feature}
          </li>
        ))}
      </ul>

      <Link
        href="/sign-up"
        data-testid="get-started-cta"
        className="inline-flex items-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
      >
        Sign up for free
      </Link>
    </div>
  );
}
