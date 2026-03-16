"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { fetchTierInfo } from "@/lib/tier-api";
import type { DeftDepartment, Tier, TierSubType } from "@/lib/tier-types";

/** Shape returned by useTier(). */
export interface UseTierReturn {
  /** Current user tier (1-6). */
  tier: Tier;
  /** Tier sub-type (e.g. "owner" for org owners, department for DEFT employees). */
  subType: TierSubType;
  /** DEFT department if tier 4. */
  deftDepartment: DeftDepartment | null;
  /** Org ID if user belongs to an org. */
  orgId: string | null;
  /** Whether the tier is still being resolved. */
  isLoading: boolean;
  /** Refresh tier info from the API. */
  refresh: () => void;
}

const TierContext = createContext<UseTierReturn | null>(null);

interface TierProviderProps {
  /** Clerk auth token (null for anonymous users). */
  token: string | null;
  children: ReactNode;
}

/** Provides tier information to the component tree via context. */
export function TierProvider({ token, children }: TierProviderProps): ReactNode {
  const [tier, setTier] = useState<Tier>(1);
  const [subType, setSubType] = useState<TierSubType>(null);
  const [deftDepartment, setDeftDepartment] = useState<DeftDepartment | null>(null);
  const [orgId, setOrgId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const mountedRef = useRef(true);

  const loadTier = useCallback(async () => {
    setIsLoading(true);
    try {
      const info = await fetchTierInfo(token);
      if (!mountedRef.current) return;
      setTier(info.tier);
      setSubType(info.sub_type);
      setDeftDepartment(info.deft_department ?? null);
      setOrgId(info.org_id ?? null);
    } catch {
      if (!mountedRef.current) return;
      // On error, default to anonymous tier.
      setTier(1);
      setSubType(null);
      setDeftDepartment(null);
      setOrgId(null);
    } finally {
      if (mountedRef.current) {
        setIsLoading(false);
      }
    }
  }, [token]);

  useEffect(() => {
    mountedRef.current = true;
    void loadTier();
    return () => {
      mountedRef.current = false;
    };
  }, [loadTier]);

  const value: UseTierReturn = {
    tier,
    subType,
    deftDepartment,
    orgId,
    isLoading,
    refresh: loadTier,
  };

  return <TierContext.Provider value={value}>{children}</TierContext.Provider>;
}

/**
 * Access the current user's tier information.
 * Must be used within a TierProvider.
 */
export function useTier(): UseTierReturn {
  const ctx = useContext(TierContext);
  if (!ctx) {
    throw new Error("useTier must be used within a TierProvider");
  }
  return ctx;
}
