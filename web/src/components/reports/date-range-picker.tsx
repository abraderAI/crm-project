"use client";

import { useRef, useState } from "react";
import { CalendarDays } from "lucide-react";

export interface DateRangePickerProps {
  /** Start date of the range. */
  from: Date;
  /** End date of the range. */
  to: Date;
  /** Called when the user selects a new range. */
  onChange: (range: { from: Date; to: Date }) => void;
}

/** Format a Date as "Mar 1, 2026". */
function formatShortDate(date: Date): string {
  return date.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

/** Format a Date as YYYY-MM-DD for the native date input. */
function toInputValue(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

/** Reusable date-range picker with popover trigger. */
export function DateRangePicker({ from, to, onChange }: DateRangePickerProps): React.ReactNode {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const label = `${formatShortDate(from)} – ${formatShortDate(to)}`;

  function handleFromChange(e: React.ChangeEvent<HTMLInputElement>): void {
    const newFrom = new Date(e.target.value + "T00:00:00");
    if (!isNaN(newFrom.getTime())) {
      onChange({ from: newFrom, to });
    }
  }

  function handleToChange(e: React.ChangeEvent<HTMLInputElement>): void {
    const newTo = new Date(e.target.value + "T00:00:00");
    if (!isNaN(newTo.getTime())) {
      onChange({ from, to: newTo });
    }
  }

  return (
    <div ref={containerRef} className="relative" data-testid="date-range-picker">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        data-testid="date-range-trigger"
        className="inline-flex items-center gap-2 rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground transition-colors hover:bg-accent/50"
      >
        <CalendarDays className="h-4 w-4 text-muted-foreground" />
        {label}
      </button>

      {open && (
        <div
          data-testid="date-range-popover"
          className="absolute left-0 top-full z-50 mt-1 flex gap-3 rounded-lg border border-border bg-background p-4 shadow-md"
        >
          <div className="flex flex-col gap-1">
            <label htmlFor="range-from" className="text-xs text-muted-foreground">
              From
            </label>
            <input
              id="range-from"
              type="date"
              data-testid="date-range-from"
              value={toInputValue(from)}
              onChange={handleFromChange}
              className="rounded-md border border-border bg-background px-2 py-1 text-sm text-foreground"
            />
          </div>
          <div className="flex flex-col gap-1">
            <label htmlFor="range-to" className="text-xs text-muted-foreground">
              To
            </label>
            <input
              id="range-to"
              type="date"
              data-testid="date-range-to"
              value={toInputValue(to)}
              onChange={handleToChange}
              className="rounded-md border border-border bg-background px-2 py-1 text-sm text-foreground"
            />
          </div>
          <div className="flex items-end">
            <button
              type="button"
              onClick={() => setOpen(false)}
              data-testid="date-range-close"
              className="rounded-md bg-primary px-3 py-1 text-sm text-primary-foreground hover:bg-primary/90"
            >
              Apply
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
