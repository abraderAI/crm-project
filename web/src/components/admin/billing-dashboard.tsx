"use client";

import { CreditCard, Receipt } from "lucide-react";
import { cn } from "@/lib/utils";
import type { BillingInfo, Invoice } from "@/lib/api-types";

export interface BillingDashboardProps {
  /** Billing information for the org. */
  billing: BillingInfo | null;
  /** Whether data is loading. */
  loading?: boolean;
}

/** Format a date string for display. */
function formatDate(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    return d.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
  } catch {
    return dateStr;
  }
}

/** Format a currency amount for display. */
function formatAmount(amount: number, currency: string): string {
  try {
    return new Intl.NumberFormat("en-US", { style: "currency", currency }).format(amount / 100);
  } catch {
    return `${(amount / 100).toFixed(2)} ${currency}`;
  }
}

/** Payment status badge color. */
function paymentStatusColor(status: string): string {
  switch (status.toLowerCase()) {
    case "active":
    case "paid":
      return "bg-green-100 text-green-800";
    case "past_due":
    case "overdue":
      return "bg-red-100 text-red-800";
    case "trialing":
    case "pending":
      return "bg-yellow-100 text-yellow-800";
    default:
      return "bg-muted text-muted-foreground";
  }
}

/** Invoice status badge. */
function invoiceStatusColor(status: string): string {
  switch (status.toLowerCase()) {
    case "paid":
      return "bg-green-100 text-green-800";
    case "open":
    case "pending":
      return "bg-yellow-100 text-yellow-800";
    case "overdue":
      return "bg-red-100 text-red-800";
    default:
      return "bg-muted text-muted-foreground";
  }
}

/** Billing dashboard displaying org tier, payment status, and invoice history. */
export function BillingDashboard({
  billing,
  loading = false,
}: BillingDashboardProps): React.ReactNode {
  return (
    <div data-testid="billing-dashboard" className="flex flex-col gap-6">
      <div className="flex items-center gap-2">
        <CreditCard className="h-5 w-5 text-muted-foreground" data-testid="billing-icon" />
        <h2 className="text-lg font-semibold text-foreground">Billing</h2>
      </div>

      {loading && (
        <div
          className="py-8 text-center text-sm text-muted-foreground"
          data-testid="billing-loading"
        >
          Loading billing information...
        </div>
      )}

      {!loading && !billing && (
        <div className="py-8 text-center text-sm text-muted-foreground" data-testid="billing-empty">
          No billing information available.
        </div>
      )}

      {!loading && billing && (
        <>
          {/* Summary cards */}
          <div className="grid gap-4 sm:grid-cols-2" data-testid="billing-summary">
            <div className="rounded-lg border border-border p-4" data-testid="billing-tier-card">
              <p className="text-xs text-muted-foreground">Plan</p>
              <p className="mt-1 text-lg font-semibold capitalize text-foreground">
                {billing.tier}
              </p>
            </div>
            <div className="rounded-lg border border-border p-4" data-testid="billing-status-card">
              <p className="text-xs text-muted-foreground">Payment Status</p>
              <span
                className={cn(
                  "mt-1 inline-block rounded-full px-2.5 py-0.5 text-sm font-medium",
                  paymentStatusColor(billing.payment_status),
                )}
                data-testid="billing-payment-status"
              >
                {billing.payment_status}
              </span>
            </div>
          </div>

          {/* Invoices */}
          <div className="flex flex-col gap-3">
            <div className="flex items-center gap-2">
              <Receipt className="h-4 w-4 text-muted-foreground" />
              <h3 className="text-sm font-semibold text-foreground">
                Invoices ({billing.invoices.length})
              </h3>
            </div>

            {billing.invoices.length === 0 ? (
              <p className="text-sm text-muted-foreground" data-testid="invoices-empty">
                No invoices yet.
              </p>
            ) : (
              <div
                className="divide-y divide-border rounded-lg border border-border"
                data-testid="invoice-list"
              >
                {billing.invoices.map((inv: Invoice) => (
                  <div
                    key={inv.id}
                    className="flex items-center gap-3 px-4 py-2.5"
                    data-testid={`invoice-item-${inv.id}`}
                  >
                    <span
                      className={cn(
                        "rounded-full px-2 py-0.5 text-xs font-medium",
                        invoiceStatusColor(inv.status),
                      )}
                      data-testid={`invoice-status-${inv.id}`}
                    >
                      {inv.status}
                    </span>
                    <span
                      className="text-sm font-medium text-foreground"
                      data-testid={`invoice-amount-${inv.id}`}
                    >
                      {formatAmount(inv.amount, inv.currency)}
                    </span>
                    <span
                      className="ml-auto text-xs text-muted-foreground"
                      data-testid={`invoice-date-${inv.id}`}
                    >
                      {formatDate(inv.issued_at)}
                    </span>
                    {inv.due_at && (
                      <span
                        className="text-xs text-muted-foreground"
                        data-testid={`invoice-due-${inv.id}`}
                      >
                        Due: {formatDate(inv.due_at)}
                      </span>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
