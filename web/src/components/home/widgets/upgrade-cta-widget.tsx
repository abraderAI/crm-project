"use client";

import type { ReactNode } from "react";
import Link from "next/link";
import { Zap } from "lucide-react";

/** Pro plan benefits shown in the CTA card. */
const PRO_BENEFITS = [
  "Organization workspace with team collaboration",
  "Priority support with dedicated SLAs",
  "Advanced reporting and analytics",
] as const;

/** "Upgrade to Pro" CTA card visible to Tier 2 users only. */
export function UpgradeCTAWidget(): ReactNode {
  return (
    <div data-testid="upgrade-cta-widget" className="space-y-3">
      <div className="flex items-center gap-2">
        <Zap className="h-5 w-5 text-yellow-500" />
        <p className="text-sm font-medium text-foreground">Upgrade to Pro</p>
      </div>

      <p className="text-sm text-muted-foreground">
        Unlock the full power of the DEFT platform for your team.
      </p>

      <ul className="space-y-1" data-testid="upgrade-benefits">
        {PRO_BENEFITS.map((benefit) => (
          <li key={benefit} className="flex items-start gap-2 text-sm text-muted-foreground">
            <span className="mt-0.5 text-yellow-500">✓</span>
            {benefit}
          </li>
        ))}
      </ul>

      <Link
        href="/upgrade"
        data-testid="upgrade-cta-link"
        className="inline-flex items-center rounded-md bg-yellow-500 px-4 py-2 text-sm font-medium text-white hover:bg-yellow-600"
      >
        Upgrade now
      </Link>
    </div>
  );
}
