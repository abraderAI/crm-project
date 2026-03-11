"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import type { TypingPayload, WSMessage } from "@/lib/api-types";

/** How long before a typing indicator expires (ms). */
const TYPING_TIMEOUT = 3000;
/** Minimum interval between sending typing events (ms). */
const TYPING_THROTTLE = 2000;

/** A user currently typing. */
export interface TypingUser {
  userId: string;
  userName: string;
  expiresAt: number;
}

/** Options for the useTyping hook. */
export interface UseTypingOptions {
  /** The thread ID to track typing for. */
  threadId: string;
  /** Current user's ID (to exclude from displayed list). */
  currentUserId?: string;
  /** Send a typing command to the WS server. */
  sendTyping?: (threadId: string) => void;
}

/** Return value of the useTyping hook. */
export interface UseTypingReturn {
  /** List of users currently typing (excluding current user). */
  typingUsers: TypingUser[];
  /** Call this when the current user types. Throttled internally. */
  handleLocalTyping: () => void;
  /** Call this to process an incoming typing WS event. */
  handleRemoteTyping: (msg: WSMessage<TypingPayload>) => void;
}

/** Hook to manage typing indicators for a thread. */
export function useTyping({
  threadId,
  currentUserId,
  sendTyping,
}: UseTypingOptions): UseTypingReturn {
  const [typingUsers, setTypingUsers] = useState<TypingUser[]>([]);
  const lastSentRef = useRef(0);
  const cleanupTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Periodically clean up expired typing entries.
  useEffect(() => {
    cleanupTimerRef.current = setInterval(() => {
      const now = Date.now();
      setTypingUsers((prev) => {
        const filtered = prev.filter((u) => u.expiresAt > now);
        // Only update if something changed.
        if (filtered.length !== prev.length) return filtered;
        return prev;
      });
    }, 1000);

    return () => {
      if (cleanupTimerRef.current) clearInterval(cleanupTimerRef.current);
    };
  }, []);

  // Reset typing users when thread changes.
  /* eslint-disable react-hooks/set-state-in-effect */
  useEffect(() => {
    setTypingUsers([]);
  }, [threadId]);
  /* eslint-enable react-hooks/set-state-in-effect */

  const handleLocalTyping = useCallback(() => {
    const now = Date.now();
    if (now - lastSentRef.current < TYPING_THROTTLE) return;
    lastSentRef.current = now;
    sendTyping?.(threadId);
  }, [threadId, sendTyping]);

  const handleRemoteTyping = useCallback(
    (msg: WSMessage<TypingPayload>) => {
      const { user_id, user_name, thread_id } = msg.payload;

      // Ignore typing events from other threads or from self.
      if (thread_id !== threadId) return;
      if (user_id === currentUserId) return;

      const expiresAt = Date.now() + TYPING_TIMEOUT;

      setTypingUsers((prev) => {
        const existing = prev.findIndex((u) => u.userId === user_id);
        if (existing >= 0) {
          const updated = [...prev];
          updated[existing] = {
            userId: user_id,
            userName: user_name ?? user_id,
            expiresAt,
          };
          return updated;
        }
        return [...prev, { userId: user_id, userName: user_name ?? user_id, expiresAt }];
      });
    },
    [threadId, currentUserId],
  );

  return { typingUsers, handleLocalTyping, handleRemoteTyping };
}
