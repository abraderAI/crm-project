"use client";

import { useCallback, useState } from "react";
import { useAuth } from "@clerk/nextjs";

import type { Flag } from "@/lib/api-types";
import { resolveFlag, dismissFlag } from "@/lib/entity-api";
import { ModerationQueue } from "./moderation-queue";

export interface ModerationViewProps {
  /** Initial flags fetched on the server. */
  initialFlags: Flag[];
}

/** Client wrapper wiring ModerationQueue to entity-api flag mutations via Clerk auth. */
export function ModerationView({ initialFlags }: ModerationViewProps): React.ReactNode {
  const { getToken } = useAuth();
  const [flags, setFlags] = useState<Flag[]>(initialFlags);

  const handleResolve = useCallback(
    async (flagId: string, note: string) => {
      const token = await getToken();
      if (!token) return;
      const updated = await resolveFlag(token, flagId, note);
      setFlags((prev) => prev.map((f) => (f.id === flagId ? updated : f)));
    },
    [getToken],
  );

  const handleDismiss = useCallback(
    async (flagId: string) => {
      const token = await getToken();
      if (!token) return;
      const updated = await dismissFlag(token, flagId);
      setFlags((prev) => prev.map((f) => (f.id === flagId ? updated : f)));
    },
    [getToken],
  );

  return <ModerationQueue flags={flags} onResolve={handleResolve} onDismiss={handleDismiss} />;
}
