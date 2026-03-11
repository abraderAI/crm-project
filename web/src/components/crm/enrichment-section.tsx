"use client";

import { Sparkles, ArrowRight, Clock } from "lucide-react";
import type { EnrichmentData } from "@/lib/crm-types";
import { formatDate } from "@/components/thread/thread-list";

export interface EnrichmentSectionProps {
  enrichment: EnrichmentData | null;
  onEnrich?: () => void;
  loading?: boolean;
}

/** AI enrichment section: summary, next action, and Enrich button. */
export function EnrichmentSection({
  enrichment,
  onEnrich,
  loading = false,
}: EnrichmentSectionProps): React.ReactNode {
  return (
    <div data-testid="enrichment-section">
      <div className="mb-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Sparkles className="h-4 w-4 text-primary" />
          <h3 className="text-sm font-semibold text-foreground">AI Enrichment</h3>
        </div>
        {onEnrich && (
          <button
            onClick={onEnrich}
            disabled={loading}
            data-testid="enrich-button"
            className="rounded-md bg-primary px-3 py-1 text-xs font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {loading ? "Enriching..." : "Enrich"}
          </button>
        )}
      </div>

      {enrichment ? (
        <div className="space-y-3" data-testid="enrichment-content">
          {/* Summary */}
          {enrichment.summary && (
            <div data-testid="enrichment-summary">
              <p className="mb-1 text-xs font-medium text-muted-foreground">Summary</p>
              <p className="text-sm text-foreground">{enrichment.summary}</p>
            </div>
          )}

          {/* Next action */}
          {enrichment.next_action && (
            <div className="flex items-start gap-2" data-testid="enrichment-next-action">
              <ArrowRight className="mt-0.5 h-3.5 w-3.5 shrink-0 text-primary" />
              <div>
                <p className="mb-0.5 text-xs font-medium text-muted-foreground">
                  Suggested next action
                </p>
                <p className="text-sm text-foreground">{enrichment.next_action}</p>
              </div>
            </div>
          )}

          {/* Enriched at */}
          {enrichment.enriched_at && (
            <div
              className="flex items-center gap-1 text-xs text-muted-foreground"
              data-testid="enrichment-timestamp"
            >
              <Clock className="h-3 w-3" />
              <span>Enriched {formatDate(enrichment.enriched_at)}</span>
            </div>
          )}
        </div>
      ) : (
        <p className="text-sm text-muted-foreground" data-testid="enrichment-empty">
          No AI enrichment data yet. Click Enrich to analyze this lead.
        </p>
      )}
    </div>
  );
}
