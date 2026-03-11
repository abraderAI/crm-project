"use client";

import { Monitor, Moon, Sun } from "lucide-react";
import { useTheme, type ThemeMode } from "./theme-provider";

const MODES: { value: ThemeMode; label: string; Icon: typeof Sun }[] = [
  { value: "light", label: "Light", Icon: Sun },
  { value: "dark", label: "Dark", Icon: Moon },
  { value: "system", label: "System", Icon: Monitor },
];

/** Cycles through light → dark → system on click. */
export function ThemeToggle(): React.ReactNode {
  const { mode, setMode } = useTheme();

  const cycle = (): void => {
    const currentIndex = MODES.findIndex((m) => m.value === mode);
    const nextIndex = (currentIndex + 1) % MODES.length;
    const next = MODES[nextIndex];
    if (next) {
      setMode(next.value);
    }
  };

  const current = MODES.find((m) => m.value === mode);
  const CurrentIcon = current?.Icon ?? Sun;

  return (
    <button
      onClick={cycle}
      aria-label={`Theme: ${mode}. Click to change.`}
      title={`Theme: ${mode}`}
      className="inline-flex items-center justify-center rounded-md p-2 text-foreground/70 transition-colors hover:bg-foreground/10 hover:text-foreground"
      data-testid="theme-toggle"
    >
      <CurrentIcon className="h-5 w-5" />
    </button>
  );
}
