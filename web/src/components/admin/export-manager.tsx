"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useAuth } from "@clerk/nextjs";
import { Download, FileDown } from "lucide-react";
import { cn } from "@/lib/utils";
import { clientMutate, buildUrl, buildHeaders, parseResponse } from "@/lib/api-client";
import type {
  AdminExport,
  AdminExportFormat,
  AdminExportStatus,
  AdminExportType,
} from "@/lib/api-types";

export interface ExportManagerProps {
  /** Initial list of exports fetched server-side. */
  initialExports: AdminExport[];
}

/** Map export status to badge color classes. */
function statusColor(status: AdminExportStatus): string {
  switch (status) {
    case "pending":
      return "bg-yellow-100 text-yellow-800";
    case "processing":
      return "bg-blue-100 text-blue-800";
    case "completed":
      return "bg-green-100 text-green-800";
    case "failed":
      return "bg-red-100 text-red-800";
  }
}

/** Format a date string for display. */
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

/** Whether an export status indicates it is still in progress. */
function isInProgress(status: AdminExportStatus): boolean {
  return status === "pending" || status === "processing";
}

/** Trigger form + polling history table for admin data exports. */
export function ExportManager({ initialExports }: ExportManagerProps): React.ReactNode {
  const { getToken } = useAuth();
  const [exports, setExports] = useState<AdminExport[]>(initialExports);
  const [exportType, setExportType] = useState<AdminExportType>("users");
  const [exportFormat, setExportFormat] = useState<AdminExportFormat>("csv");
  const [submitting, setSubmitting] = useState(false);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  /** Poll a single export by ID and update state. */
  const pollExport = useCallback(
    async (id: string) => {
      const token = await getToken();
      if (!token) return;
      const url = buildUrl(`/admin/exports/${id}`);
      const response = await fetch(url, {
        method: "GET",
        headers: buildHeaders(token),
      });
      const updated = await parseResponse<AdminExport>(response);
      setExports((prev) => prev.map((e) => (e.id === id ? updated : e)));
    },
    [getToken],
  );

  /** Start/stop polling based on whether any exports are in progress. */
  useEffect(() => {
    const hasPending = exports.some((e) => isInProgress(e.status));

    if (hasPending && !intervalRef.current) {
      intervalRef.current = setInterval(() => {
        const inProgress = exports.filter((e) => isInProgress(e.status));
        for (const exp of inProgress) {
          void pollExport(exp.id);
        }
      }, 3000);
    }

    if (!hasPending && intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [exports, pollExport]);

  /** Handle create export form submission. */
  const handleCreate = useCallback(async () => {
    setSubmitting(true);
    try {
      const token = await getToken();
      if (!token) return;

      const result = await clientMutate<{ export_id: string; status: string }>(
        "POST",
        "/admin/exports",
        {
          token,
          body: { type: exportType, format: exportFormat },
        },
      );

      // Add the new export to the list with pending status.
      const newExport: AdminExport = {
        id: result.export_id,
        type: exportType,
        filters: "{}",
        format: exportFormat,
        status: "pending",
        requested_by: "",
        created_at: new Date().toISOString(),
      };
      setExports((prev) => [newExport, ...prev]);
    } finally {
      setSubmitting(false);
    }
  }, [getToken, exportType, exportFormat]);

  /** Build a download URL for a completed export. */
  const downloadUrl = (filePath: string): string => {
    return buildUrl(`/admin/exports/download/${filePath}`);
  };

  return (
    <div data-testid="export-manager" className="flex flex-col gap-6">
      <div className="flex items-center gap-2">
        <FileDown className="h-5 w-5 text-muted-foreground" />
        <h2 className="text-lg font-semibold text-foreground">Data Exports</h2>
      </div>

      {/* Trigger form */}
      <div
        className="flex flex-wrap items-end gap-3 rounded-lg border border-border p-4"
        data-testid="export-form"
      >
        <div className="flex flex-col gap-1">
          <label htmlFor="export-type" className="text-xs font-medium text-muted-foreground">
            Type
          </label>
          <select
            id="export-type"
            data-testid="export-type-select"
            value={exportType}
            onChange={(e) => setExportType(e.target.value as AdminExportType)}
            className="rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground"
          >
            <option value="users">Users</option>
            <option value="orgs">Organizations</option>
            <option value="audit">Audit Log</option>
          </select>
        </div>

        <div className="flex flex-col gap-1">
          <label htmlFor="export-format" className="text-xs font-medium text-muted-foreground">
            Format
          </label>
          <select
            id="export-format"
            data-testid="export-format-select"
            value={exportFormat}
            onChange={(e) => setExportFormat(e.target.value as AdminExportFormat)}
            className="rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground"
          >
            <option value="csv">CSV</option>
            <option value="json">JSON</option>
          </select>
        </div>

        <button
          data-testid="export-create-btn"
          onClick={handleCreate}
          disabled={submitting}
          className="rounded-md bg-primary px-4 py-1.5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
        >
          {submitting ? "Creating…" : "Create Export"}
        </button>
      </div>

      {/* Export history */}
      {exports.length === 0 ? (
        <div className="py-8 text-center text-sm text-muted-foreground" data-testid="export-empty">
          No exports yet. Create one above to get started.
        </div>
      ) : (
        <div
          className="divide-y divide-border rounded-lg border border-border"
          data-testid="export-table"
        >
          {/* Header */}
          <div className="grid grid-cols-6 gap-2 bg-muted/30 px-4 py-2 text-xs font-medium text-muted-foreground">
            <span>ID</span>
            <span>Type</span>
            <span>Format</span>
            <span>Status</span>
            <span>Created</span>
            <span>Action</span>
          </div>

          {exports.map((exp) => (
            <div
              key={exp.id}
              data-testid={`export-row-${exp.id}`}
              className="grid grid-cols-6 items-center gap-2 px-4 py-2.5 text-sm"
            >
              <span
                className="truncate font-mono text-xs text-foreground"
                data-testid={`export-id-${exp.id}`}
              >
                {exp.id}
              </span>
              <span data-testid={`export-type-${exp.id}`} className="text-foreground">
                {exp.type}
              </span>
              <span data-testid={`export-format-${exp.id}`} className="text-foreground">
                {exp.format}
              </span>
              <span
                data-testid={`export-status-${exp.id}`}
                className={cn(
                  "inline-block w-fit rounded-full px-2 py-0.5 text-xs font-medium",
                  statusColor(exp.status),
                )}
              >
                {exp.status}
              </span>
              <span className="text-xs text-muted-foreground" data-testid={`export-date-${exp.id}`}>
                {formatDate(exp.created_at)}
              </span>
              <span>
                {exp.status === "completed" && exp.file_path && (
                  <a
                    href={downloadUrl(exp.file_path)}
                    data-testid={`export-download-${exp.id}`}
                    className="inline-flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs text-foreground transition-colors hover:bg-accent"
                    download
                  >
                    <Download className="h-3 w-3" />
                    Download
                  </a>
                )}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
