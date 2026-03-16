"use client";

import { useCallback, useEffect, useState } from "react";
import { Phone, Search, ShoppingCart, AlertTriangle, X } from "lucide-react";
import type { PhoneNumber, PhoneNumberSearchResult } from "@/lib/api-types";
import { buildHeaders, buildUrl } from "@/lib/api-client";

/* ------------------------------------------------------------------ */
/*  Types                                                             */
/* ------------------------------------------------------------------ */

export interface PhoneNumberManagerProps {
  /** Organization ID for API calls. */
  org: string;
  /** Optional auth token for client-side requests. */
  token?: string | null;
}

/* ------------------------------------------------------------------ */
/*  Helpers                                                           */
/* ------------------------------------------------------------------ */

/** Format an ISO date string for display. */
function formatDate(dateStr: string): string {
  try {
    return new Date(dateStr).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  } catch {
    return dateStr;
  }
}

/* ------------------------------------------------------------------ */
/*  Component                                                         */
/* ------------------------------------------------------------------ */

/** LiveKit phone number CRUD — owned numbers table, search, and purchase flow. */
export function PhoneNumberManager({ org, token }: PhoneNumberManagerProps): React.ReactNode {
  const [ownedNumbers, setOwnedNumbers] = useState<PhoneNumber[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchResults, setSearchResults] = useState<PhoneNumberSearchResult[]>([]);
  const [searching, setSearching] = useState(false);
  const [areaCode, setAreaCode] = useState("");
  const [country, setCountry] = useState("US");
  const [purchaseTarget, setPurchaseTarget] = useState<string | null>(null);
  const [purchasing, setPurchasing] = useState(false);
  const [purchaseError, setPurchaseError] = useState("");

  const basePath = `/orgs/${org}/channels/voice/numbers`;

  /** Fetch owned phone numbers. */
  const fetchOwned = useCallback(async (): Promise<void> => {
    setLoading(true);
    try {
      const url = buildUrl(basePath);
      const res = await fetch(url, {
        method: "GET",
        headers: buildHeaders(token),
      });
      if (res.ok) {
        const json = (await res.json()) as { data: PhoneNumber[] };
        setOwnedNumbers(json.data);
      }
    } finally {
      setLoading(false);
    }
  }, [basePath, token]);

  useEffect(() => {
    void fetchOwned();
  }, [fetchOwned]);

  /** Search for available phone numbers. */
  const handleSearch = async (): Promise<void> => {
    setSearching(true);
    setPurchaseError("");
    try {
      const url = buildUrl(`${basePath}/search`);
      const res = await fetch(url, {
        method: "POST",
        headers: buildHeaders(token),
        body: JSON.stringify({ area_code: areaCode, country }),
      });
      if (res.ok) {
        const json = (await res.json()) as { data: PhoneNumberSearchResult[] };
        setSearchResults(json.data);
      }
    } finally {
      setSearching(false);
    }
  };

  /** Purchase a phone number after confirmation. */
  const handlePurchase = async (): Promise<void> => {
    if (!purchaseTarget) return;
    setPurchasing(true);
    setPurchaseError("");
    try {
      const url = buildUrl(`${basePath}/purchase`);
      const res = await fetch(url, {
        method: "POST",
        headers: buildHeaders(token),
        body: JSON.stringify({ phone_number: purchaseTarget }),
      });
      if (!res.ok) {
        setPurchaseError("Failed to purchase phone number. Please try again.");
        return;
      }
      // Success — close modal, refresh owned numbers, clear search results.
      setPurchaseTarget(null);
      setSearchResults((prev) => prev.filter((r) => r.phone_number !== purchaseTarget));
      void fetchOwned();
    } catch {
      setPurchaseError("Failed to purchase phone number. Please try again.");
    } finally {
      setPurchasing(false);
    }
  };

  return (
    <div data-testid="phone-number-manager" className="flex flex-col gap-6">
      <h2 className="text-lg font-semibold text-foreground">Phone Number Management</h2>

      {/* ── Owned Numbers ─────────────────────────────────── */}
      <section className="rounded-lg border border-border p-4">
        <h3 className="text-sm font-semibold text-foreground">Owned Numbers</h3>

        {loading && (
          <p className="mt-2 text-sm text-muted-foreground" data-testid="owned-numbers-loading">
            Loading…
          </p>
        )}

        {!loading && ownedNumbers.length === 0 && (
          <p className="mt-2 text-sm text-muted-foreground" data-testid="owned-numbers-empty">
            No phone numbers owned yet.
          </p>
        )}

        {!loading && ownedNumbers.length > 0 && (
          <table className="mt-2 w-full text-sm" data-testid="owned-numbers-table">
            <thead>
              <tr className="border-b border-border text-left text-xs text-muted-foreground">
                <th className="pb-2 pr-4 font-medium">Number</th>
                <th className="pb-2 pr-4 font-medium">Status</th>
                <th className="pb-2 pr-4 font-medium">Dispatch Rule</th>
                <th className="pb-2 font-medium">Purchased</th>
              </tr>
            </thead>
            <tbody>
              {ownedNumbers.map((num) => (
                <tr
                  key={num.phone_number}
                  className="border-b border-border last:border-0"
                  data-testid={`owned-row-${num.phone_number}`}
                >
                  <td className="py-2 pr-4 font-mono text-foreground">
                    <Phone className="mr-1 inline-block h-3 w-3" />
                    {num.phone_number}
                  </td>
                  <td className="py-2 pr-4 text-foreground">{num.status}</td>
                  <td className="py-2 pr-4 text-foreground">{num.dispatch_rule_id}</td>
                  <td className="py-2 text-muted-foreground">{formatDate(num.purchased_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      {/* ── Search Available Numbers ──────────────────────── */}
      <section className="rounded-lg border border-border p-4">
        <h3 className="text-sm font-semibold text-foreground">Search Available Numbers</h3>

        <div className="mt-3 flex items-end gap-3">
          <div className="flex flex-col gap-1">
            <label htmlFor="search-area-code" className="text-xs font-medium text-muted-foreground">
              Area Code
            </label>
            <input
              id="search-area-code"
              type="text"
              value={areaCode}
              onChange={(e) => setAreaCode(e.target.value)}
              placeholder="e.g. 415"
              data-testid="search-area-code"
              className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
            />
          </div>

          <div className="flex flex-col gap-1">
            <label htmlFor="search-country" className="text-xs font-medium text-muted-foreground">
              Country
            </label>
            <select
              id="search-country"
              value={country}
              onChange={(e) => setCountry(e.target.value)}
              data-testid="search-country"
              className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
            >
              <option value="US">United States</option>
              <option value="CA">Canada</option>
              <option value="GB">United Kingdom</option>
            </select>
          </div>

          <button
            type="button"
            onClick={() => void handleSearch()}
            disabled={searching}
            data-testid="search-btn"
            className="inline-flex items-center gap-1 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            <Search className="h-4 w-4" />
            {searching ? "Searching…" : "Search"}
          </button>
        </div>

        {/* Search results */}
        {searchResults.length > 0 && (
          <table className="mt-4 w-full text-sm" data-testid="search-results-table">
            <thead>
              <tr className="border-b border-border text-left text-xs text-muted-foreground">
                <th className="pb-2 pr-4 font-medium">Number</th>
                <th className="pb-2 pr-4 font-medium">Country</th>
                <th className="pb-2 pr-4 font-medium">Area Code</th>
                <th className="pb-2 pr-4 font-medium">Monthly Cost</th>
                <th className="pb-2 font-medium">Action</th>
              </tr>
            </thead>
            <tbody>
              {searchResults.map((result) => (
                <tr
                  key={result.phone_number}
                  className="border-b border-border last:border-0"
                  data-testid={`search-row-${result.phone_number}`}
                >
                  <td className="py-2 pr-4 font-mono text-foreground">{result.phone_number}</td>
                  <td className="py-2 pr-4 text-foreground">{result.country}</td>
                  <td className="py-2 pr-4 text-foreground">{result.area_code}</td>
                  <td className="py-2 pr-4 text-foreground">{result.monthly_cost}</td>
                  <td className="py-2">
                    <button
                      type="button"
                      onClick={() => {
                        setPurchaseTarget(result.phone_number);
                        setPurchaseError("");
                      }}
                      data-testid="purchase-btn"
                      className="inline-flex items-center gap-1 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90"
                    >
                      <ShoppingCart className="h-3 w-3" />
                      Purchase
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      {/* ── Purchase Confirmation Modal ───────────────────── */}
      {purchaseTarget && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
          data-testid="purchase-confirm-modal"
          role="dialog"
          aria-modal="true"
          aria-label="Confirm phone number purchase"
        >
          <div className="w-full max-w-md rounded-lg border border-border bg-background p-6 shadow-lg">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-semibold text-foreground">Confirm Purchase</h3>
              <button
                type="button"
                onClick={() => {
                  setPurchaseTarget(null);
                  setPurchaseError("");
                }}
                data-testid="purchase-modal-close"
                className="text-muted-foreground hover:text-foreground"
                aria-label="Close"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="mt-4 flex items-start gap-3 rounded-md border border-yellow-300 bg-yellow-50 p-3">
              <AlertTriangle className="mt-0.5 h-5 w-5 flex-shrink-0 text-yellow-600" />
              <p className="text-sm text-yellow-800">
                This will add a phone number to your account and may incur charges. This is a
                billable action.
              </p>
            </div>

            <p className="mt-4 text-sm text-foreground">
              You are about to purchase: <strong>{purchaseTarget}</strong>
            </p>

            {purchaseError && (
              <p className="mt-2 text-xs text-destructive" data-testid="purchase-error">
                {purchaseError}
              </p>
            )}

            <div className="mt-6 flex justify-end gap-3">
              <button
                type="button"
                onClick={() => {
                  setPurchaseTarget(null);
                  setPurchaseError("");
                }}
                data-testid="purchase-cancel-btn"
                className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-accent"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => void handlePurchase()}
                disabled={purchasing}
                data-testid="purchase-confirm-btn"
                className="inline-flex items-center gap-1 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                <ShoppingCart className="h-4 w-4" />
                {purchasing ? "Purchasing…" : "Confirm Purchase"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
