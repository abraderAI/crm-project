"use client";

import { useCallback, useState, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@clerk/nextjs";
import { Check, Zap } from "lucide-react";

import { useTier } from "@/hooks/use-tier";
import { upgradeToCustomer } from "@/lib/upgrade-api";

/** Features included in the Developer (Tier 2) plan. */
const DEVELOPER_FEATURES = [
  "Personal account",
  "Community forum access",
  "Basic support tickets",
  "Public API access",
] as const;

/** Features included in the Customer (Tier 3) plan. */
const CUSTOMER_FEATURES = [
  "Everything in Developer",
  "Organization workspace",
  "Team collaboration",
  "Priority support with SLAs",
  "Advanced reporting & analytics",
  "Custom integrations",
  "CRM pipeline access",
] as const;

/** Feature list renderer. */
function FeatureList({
  features,
  testId,
}: {
  features: readonly string[];
  testId: string;
}): ReactNode {
  return (
    <ul className="space-y-2" data-testid={testId}>
      {features.map((feature) => (
        <li key={feature} className="flex items-start gap-2 text-sm text-muted-foreground">
          <Check className="mt-0.5 h-4 w-4 shrink-0 text-green-500" />
          {feature}
        </li>
      ))}
    </ul>
  );
}

/** Self-service upgrade page: Tier 2 → Tier 3 conversion flow. */
export function UpgradePage(): ReactNode {
  const router = useRouter();
  const { getToken } = useAuth();
  const { refresh } = useTier();
  const [isUpgrading, setIsUpgrading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleUpgrade = useCallback(async () => {
    setError(null);
    setIsUpgrading(true);
    try {
      const token = await getToken();
      if (!token) {
        setIsUpgrading(false);
        return;
      }
      await upgradeToCustomer(token);
      refresh();
      router.push("/");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Upgrade failed";
      setError(message);
      setIsUpgrading(false);
    }
  }, [getToken, refresh, router]);

  return (
    <div className="mx-auto max-w-4xl px-6 py-10" data-testid="upgrade-page">
      <h1 className="text-2xl font-bold text-foreground">Upgrade Your Plan</h1>
      <p className="mt-2 text-muted-foreground">
        Compare plans and unlock the full power of the DEFT platform.
      </p>

      <div className="mt-8 grid gap-6 md:grid-cols-2">
        {/* Developer (Tier 2) - Current */}
        <div
          data-testid="plan-card-developer"
          className="rounded-lg border border-border bg-card p-6"
        >
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-foreground">Developer</h2>
            <span className="rounded-full bg-muted px-3 py-1 text-xs font-medium text-muted-foreground">
              Current Plan
            </span>
          </div>
          <p className="mt-1 text-sm text-muted-foreground">Tier 2 — Registered Developer</p>
          <p className="mt-4 text-2xl font-bold text-foreground">Free</p>
          <div className="mt-6">
            <FeatureList features={DEVELOPER_FEATURES} testId="developer-features" />
          </div>
        </div>

        {/* Customer (Tier 3) - Target */}
        <div
          data-testid="plan-card-customer"
          className="rounded-lg border-2 border-yellow-500 bg-card p-6"
        >
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-foreground">Customer</h2>
            <span className="rounded-full bg-yellow-100 px-3 py-1 text-xs font-medium text-yellow-700">
              Recommended
            </span>
          </div>
          <p className="mt-1 text-sm text-muted-foreground">Tier 3 — Paying Customer</p>
          <p className="mt-4 text-2xl font-bold text-foreground">
            Trial{" "}
            <span className="text-sm font-normal text-muted-foreground">(billing deferred)</span>
          </p>
          <div className="mt-6">
            <FeatureList features={CUSTOMER_FEATURES} testId="customer-features" />
          </div>

          <button
            data-testid="upgrade-button"
            onClick={handleUpgrade}
            disabled={isUpgrading}
            className="mt-6 inline-flex w-full items-center justify-center gap-2 rounded-md bg-yellow-500 px-4 py-2.5 text-sm font-medium text-white hover:bg-yellow-600 disabled:opacity-50"
          >
            <Zap className="h-4 w-4" />
            {isUpgrading ? "Upgrading…" : "Activate Trial"}
          </button>
        </div>
      </div>

      {error && (
        <div
          data-testid="upgrade-error"
          className="mt-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
        >
          {error}
        </div>
      )}
    </div>
  );
}
