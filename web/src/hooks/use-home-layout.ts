"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { getDefaultLayout } from "@/lib/default-layouts";
import { fetchHomePreferences, saveHomePreferences } from "@/lib/tier-api";
import type { DeftDepartment, Tier, WidgetConfig } from "@/lib/tier-types";

/** Return value of the useHomeLayout hook. */
export interface UseHomeLayoutReturn {
  /** Current ordered list of widget configs. */
  layout: WidgetConfig[];
  /** Whether layout is still loading. */
  isLoading: boolean;
  /** Whether the user has customized their layout. */
  isCustomized: boolean;
  /** Update the layout (locally and persist to server). */
  updateLayout: (newLayout: WidgetConfig[]) => Promise<void>;
  /** Reset to the default tier layout. */
  resetToDefault: () => Promise<void>;
}

/**
 * Manages home screen widget layout for a given tier.
 * Fetches saved preferences; falls back to the static default layout for the tier.
 */
export function useHomeLayout(
  tier: Tier,
  token: string | null,
  department?: DeftDepartment | null,
): UseHomeLayoutReturn {
  const [layout, setLayout] = useState<WidgetConfig[]>(() => getDefaultLayout(tier, department));
  const [isLoading, setIsLoading] = useState(true);
  const [isCustomized, setIsCustomized] = useState(false);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    let active = true;

    async function load(): Promise<void> {
      setIsLoading(true);
      try {
        if (token) {
          const prefs = await fetchHomePreferences(token);
          if (!active) return;
          if (prefs?.layout && prefs.layout.length > 0) {
            setLayout(prefs.layout);
            setIsCustomized(true);
          } else {
            setLayout(getDefaultLayout(tier, department));
            setIsCustomized(false);
          }
        } else {
          setLayout(getDefaultLayout(tier, department));
          setIsCustomized(false);
        }
      } catch {
        if (!active) return;
        setLayout(getDefaultLayout(tier, department));
        setIsCustomized(false);
      } finally {
        if (active) {
          setIsLoading(false);
        }
      }
    }

    void load();
    return () => {
      active = false;
      mountedRef.current = false;
    };
  }, [tier, token, department]);

  const updateLayout = useCallback(
    async (newLayout: WidgetConfig[]): Promise<void> => {
      setLayout(newLayout);
      if (token) {
        await saveHomePreferences(token, newLayout);
        if (mountedRef.current) {
          setIsCustomized(true);
        }
      }
    },
    [token],
  );

  const resetToDefault = useCallback(async (): Promise<void> => {
    const defaultLayout = getDefaultLayout(tier, department);
    setLayout(defaultLayout);
    setIsCustomized(false);
    if (token) {
      await saveHomePreferences(token, defaultLayout);
    }
  }, [tier, department, token]);

  return { layout, isLoading, isCustomized, updateLayout, resetToDefault };
}
