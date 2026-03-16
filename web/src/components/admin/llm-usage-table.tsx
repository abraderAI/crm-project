"use client";

import { BrainCircuit } from "lucide-react";
import type { LlmUsageEntry } from "@/lib/api-types";

export interface LlmUsageTableProps {
  /** LLM usage log entries. */
  entries: LlmUsageEntry[];
  /** Whether data is loading. */
  loading?: boolean;
}

/** Format a date string for table display. */
function formatDate(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    return d.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return dateStr;
  }
}

/** LLM usage log table displaying model, tokens, latency, and timestamps. */
export function LlmUsageTable({ entries, loading = false }: LlmUsageTableProps): React.ReactNode {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <BrainCircuit className="h-5 w-5 text-muted-foreground" />
        <h2 className="text-lg font-semibold text-foreground">LLM Usage Log</h2>
      </div>

      {loading && (
        <div
          className="py-8 text-center text-sm text-muted-foreground"
          data-testid="llm-usage-loading"
        >
          Loading LLM usage log…
        </div>
      )}

      {!loading && entries.length === 0 && (
        <div
          className="py-8 text-center text-sm text-muted-foreground"
          data-testid="llm-usage-empty"
        >
          No LLM usage entries found.
        </div>
      )}

      {!loading && entries.length > 0 && (
        <div
          className="overflow-x-auto rounded-lg border border-border"
          data-testid="llm-usage-table"
        >
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/40">
                <th className="px-4 py-2 text-left font-medium text-muted-foreground">Endpoint</th>
                <th className="px-4 py-2 text-left font-medium text-muted-foreground">Model</th>
                <th className="px-4 py-2 text-right font-medium text-muted-foreground">
                  Input Tokens
                </th>
                <th className="px-4 py-2 text-right font-medium text-muted-foreground">
                  Output Tokens
                </th>
                <th className="px-4 py-2 text-right font-medium text-muted-foreground">Latency</th>
                <th className="px-4 py-2 text-right font-medium text-muted-foreground">
                  Timestamp
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {entries.map((entry) => (
                <tr key={entry.id} data-testid={`llm-usage-row-${entry.id}`}>
                  <td
                    className="px-4 py-2 font-mono text-xs text-foreground"
                    data-testid={`llm-usage-endpoint-${entry.id}`}
                  >
                    {entry.endpoint}
                  </td>
                  <td className="px-4 py-2" data-testid={`llm-usage-model-${entry.id}`}>
                    <span className="rounded-full bg-purple-100 px-2 py-0.5 text-xs font-medium text-purple-800">
                      {entry.model}
                    </span>
                  </td>
                  <td
                    className="px-4 py-2 text-right text-foreground"
                    data-testid={`llm-usage-input-${entry.id}`}
                  >
                    {entry.input_tokens.toLocaleString()}
                  </td>
                  <td
                    className="px-4 py-2 text-right text-foreground"
                    data-testid={`llm-usage-output-${entry.id}`}
                  >
                    {entry.output_tokens.toLocaleString()}
                  </td>
                  <td
                    className="px-4 py-2 text-right text-foreground"
                    data-testid={`llm-usage-latency-${entry.id}`}
                  >
                    {entry.duration_ms.toLocaleString()} ms
                  </td>
                  <td
                    className="px-4 py-2 text-right text-xs text-muted-foreground"
                    data-testid={`llm-usage-time-${entry.id}`}
                  >
                    {formatDate(entry.created_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
