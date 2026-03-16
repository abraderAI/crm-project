"use client";

import type { ReactNode } from "react";
import Link from "next/link";
import { CreditCard, Calendar, ArrowUpRight } from "lucide-react";

interface BillingStatusWidgetProps {
  /** Whether the user is an org owner. Only owners see this widget. */
  isOwner: boolean;
}

/**
 * Displays current plan, renewal date (stub), and upgrade link.
 * Visible to org owner only. Shows placeholder data pending billing integration.
 */
export function BillingStatusWidget({ isOwner }: BillingStatusWidgetProps): ReactNode {
  if (!isOwner) {
    return null;
  }

  return (
    <div data-testid="billing-status-widget" className="space-y-3">
      <div className="flex items-center gap-2">
        <CreditCard className="h-4 w-4 text-primary" />
        <span className="text-sm font-medium text-foreground" data-testid="billing-plan">
          Pro Plan
        </span>
      </div>

      <div className="flex items-center gap-1 text-xs text-muted-foreground">
        <Calendar className="h-3 w-3" />
        <span data-testid="billing-renewal">Renews: April 15, 2026</span>
      </div>

      <Link
        href="/admin/billing"
        data-testid="billing-manage-link"
        className="inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline"
      >
        Manage billing
        <ArrowUpRight className="h-3 w-3" />
      </Link>
    </div>
  );
}
