"use client";

import { createContext, useCallback, useContext, useEffect, useRef, useState } from "react";

export type ThemeMode = "light" | "dark" | "system";

interface ThemeContextValue {
  /** User's preferred mode (may be "system"). */
  mode: ThemeMode;
  /** Resolved actual theme — always "light" or "dark". */
  resolvedTheme: "light" | "dark";
  /** Change the theme mode. */
  setMode: (mode: ThemeMode) => void;
  /** Apply per-org overrides (CSS custom properties). */
  applyOrgTheme: (overrides: Record<string, string>) => void;
}

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined);

const STORAGE_KEY = "deft-theme";

function getSystemTheme(): "light" | "dark" {
  if (typeof window === "undefined") return "light";
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function resolveTheme(mode: ThemeMode): "light" | "dark" {
  if (mode === "system") return getSystemTheme();
  return mode;
}

function getStoredMode(): ThemeMode {
  if (typeof window === "undefined") return "system";
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === "light" || stored === "dark" || stored === "system") {
    return stored;
  }
  return "system";
}

export function ThemeProvider({ children }: { children: React.ReactNode }): React.ReactNode {
  const [mode, setModeState] = useState<ThemeMode>(() => getStoredMode());
  const [resolvedTheme, setResolvedTheme] = useState<"light" | "dark">(() =>
    resolveTheme(getStoredMode()),
  );
  const mountedRef = useRef(false);

  // Listen for system theme changes when mode is "system".
  useEffect(() => {
    if (mode !== "system") return;
    const mql = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = (e: MediaQueryListEvent): void => {
      setResolvedTheme(e.matches ? "dark" : "light");
    };
    mql.addEventListener("change", handler);
    return () => mql.removeEventListener("change", handler);
  }, [mode]);

  // Apply or remove the "dark" class on <html>.
  useEffect(() => {
    mountedRef.current = true;
    const root = document.documentElement;
    if (resolvedTheme === "dark") {
      root.classList.add("dark");
    } else {
      root.classList.remove("dark");
    }
  }, [resolvedTheme]);

  const setMode = useCallback((newMode: ThemeMode) => {
    setModeState(newMode);
    setResolvedTheme(resolveTheme(newMode));
    localStorage.setItem(STORAGE_KEY, newMode);
  }, []);

  const applyOrgTheme = useCallback((overrides: Record<string, string>) => {
    const root = document.documentElement;
    for (const [key, value] of Object.entries(overrides)) {
      root.style.setProperty(key, value);
    }
  }, []);

  return (
    <ThemeContext.Provider value={{ mode, resolvedTheme, setMode, applyOrgTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

/** Hook to access theme state and controls. */
export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be used within a ThemeProvider");
  }
  return ctx;
}
