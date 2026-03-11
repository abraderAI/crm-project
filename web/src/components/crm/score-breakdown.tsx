"use client";

import { CheckCircle, XCircle, TrendingUp } from "lucide-react";
import type { ScoreBreakdown } from "@/lib/crm-types";

export interface ScoreBreakdownProps {
  breakdown: ScoreBreakdown;
}

/** Displays the lead score with a breakdown of contributing rules. */
export function ScoreBreakdownView({ breakdown }: ScoreBreakdownProps): React.ReactNode {
  const matchedRules = breakdown.rules.filter((r) => r.matched);
  const unmatchedRules = breakdown.rules.filter((r) => !r.matched);

  return (
    <div data-testid="score-breakdown">
      {/* Score header */}
      <div className="mb-3 flex items-center gap-2">
        <TrendingUp className="h-5 w-5 text-primary" />
        <span className="text-lg font-bold text-foreground" data-testid="score-total">
          {breakdown.total}
        </span>
        <span className="text-sm text-muted-foreground">points</span>
      </div>

      {/* Matched rules */}
      {matchedRules.length > 0 && (
        <div className="mb-2" data-testid="score-matched-rules">
          <p className="mb-1 text-xs font-medium text-muted-foreground">Contributing rules</p>
          <div className="space-y-1">
            {matchedRules.map((rule) => (
              <div
                key={rule.name}
                className="flex items-start gap-2 rounded-md bg-green-50 p-2 dark:bg-green-900/20"
                data-testid={`score-rule-${rule.name}`}
              >
                <CheckCircle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-green-600 dark:text-green-400" />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-medium text-foreground">{rule.name}</span>
                    <span
                      className="text-xs font-semibold text-green-700 dark:text-green-300"
                      data-testid={`score-rule-points-${rule.name}`}
                    >
                      +{rule.points}
                    </span>
                  </div>
                  <p className="text-xs text-muted-foreground">{rule.description}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Unmatched rules */}
      {unmatchedRules.length > 0 && (
        <div data-testid="score-unmatched-rules">
          <p className="mb-1 text-xs font-medium text-muted-foreground">Unmatched rules</p>
          <div className="space-y-1">
            {unmatchedRules.map((rule) => (
              <div
                key={rule.name}
                className="flex items-start gap-2 rounded-md bg-muted/50 p-2"
                data-testid={`score-rule-${rule.name}`}
              >
                <XCircle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-medium text-muted-foreground">{rule.name}</span>
                    <span
                      className="text-xs text-muted-foreground"
                      data-testid={`score-rule-points-${rule.name}`}
                    >
                      +{rule.points}
                    </span>
                  </div>
                  <p className="text-xs text-muted-foreground">{rule.description}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Empty state */}
      {breakdown.rules.length === 0 && (
        <p className="text-xs text-muted-foreground" data-testid="score-no-rules">
          No scoring rules configured.
        </p>
      )}
    </div>
  );
}
