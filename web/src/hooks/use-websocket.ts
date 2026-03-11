"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import type { WSClientCommand, WSEventType, WSMessage } from "@/lib/api-types";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

/** Derive WS URL from the API base. */
function getWsUrl(token: string): string {
  const base = API_BASE_URL.replace(/^http/, "ws");
  return `${base}/v1/ws?token=${encodeURIComponent(token)}`;
}

/** Connection states for the WebSocket. */
export type WSConnectionState = "connecting" | "connected" | "disconnected" | "reconnecting";

/** Event handler map keyed by WSEventType. */
export type WSEventHandlers = Partial<Record<WSEventType, (msg: WSMessage) => void>>;

/** Options for useWebSocket. */
export interface UseWebSocketOptions {
  /** JWT token for authentication. */
  token: string | null;
  /** Whether to enable the connection (e.g. false during SSR). */
  enabled?: boolean;
  /** Event handlers keyed by event type. */
  onEvent?: WSEventHandlers;
  /** Called on any connection error. */
  onError?: (error: Event) => void;
  /** Called when connection state changes. */
  onStateChange?: (state: WSConnectionState) => void;
}

/** Return value of the useWebSocket hook. */
export interface UseWebSocketReturn {
  /** Current connection state. */
  state: WSConnectionState;
  /** Send a typed command to the server. */
  send: (command: WSClientCommand) => void;
  /** Subscribe to a channel. */
  subscribe: (channel: string) => void;
  /** Unsubscribe from a channel. */
  unsubscribe: (channel: string) => void;
  /** Last received message (any type). */
  lastMessage: WSMessage | null;
}

const BASE_RECONNECT_DELAY = 1000;
const MAX_RECONNECT_DELAY = 30000;
const PING_INTERVAL = 30000;

/** WebSocket client hook with auth, auto-reconnect, and typed event dispatch. */
export function useWebSocket({
  token,
  enabled = true,
  onEvent,
  onError,
  onStateChange,
}: UseWebSocketOptions): UseWebSocketReturn {
  const [state, setState] = useState<WSConnectionState>("disconnected");
  const [lastMessage, setLastMessage] = useState<WSMessage | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectAttemptRef = useRef(0);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pingTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const subscribedChannelsRef = useRef<Set<string>>(new Set());
  const mountedRef = useRef(true);
  const onEventRef = useRef(onEvent);
  const onErrorRef = useRef(onError);
  const onStateChangeRef = useRef(onStateChange);
  const tokenRef = useRef(token);
  const connectRef = useRef<() => void>(() => {});

  // Keep refs current without re-triggering effects.
  useEffect(() => {
    onEventRef.current = onEvent;
  }, [onEvent]);
  useEffect(() => {
    onErrorRef.current = onError;
  }, [onError]);
  useEffect(() => {
    onStateChangeRef.current = onStateChange;
  }, [onStateChange]);
  useEffect(() => {
    tokenRef.current = token;
  }, [token]);

  const updateState = useCallback((newState: WSConnectionState) => {
    if (!mountedRef.current) return;
    setState(newState);
    onStateChangeRef.current?.(newState);
  }, []);

  const clearTimers = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
    if (pingTimerRef.current) {
      clearInterval(pingTimerRef.current);
      pingTimerRef.current = null;
    }
  }, []);

  const startPing = useCallback(() => {
    if (pingTimerRef.current) clearInterval(pingTimerRef.current);
    pingTimerRef.current = setInterval(() => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({ action: "ping" }));
      }
    }, PING_INTERVAL);
  }, []);

  const send = useCallback((command: WSClientCommand) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(command));
    }
  }, []);

  const subscribe = useCallback(
    (channel: string) => {
      subscribedChannelsRef.current.add(channel);
      send({ action: "subscribe", channel });
    },
    [send],
  );

  const unsubscribe = useCallback(
    (channel: string) => {
      subscribedChannelsRef.current.delete(channel);
      send({ action: "unsubscribe", channel });
    },
    [send],
  );

  /** Resubscribe to all tracked channels (after reconnect). */
  const resubscribeAll = useCallback(() => {
    for (const channel of subscribedChannelsRef.current) {
      send({ action: "subscribe", channel });
    }
  }, [send]);

  // Use a ref-based connect to avoid circular useCallback deps.
  connectRef.current = () => {
    const tok = tokenRef.current;
    if (!tok || !mountedRef.current) return;

    clearTimers();
    updateState("connecting");

    const ws = new WebSocket(getWsUrl(tok));
    wsRef.current = ws;

    ws.onopen = () => {
      if (!mountedRef.current) return;
      reconnectAttemptRef.current = 0;
      updateState("connected");
      startPing();
      resubscribeAll();
    };

    ws.onmessage = (event: MessageEvent) => {
      if (!mountedRef.current) return;
      try {
        const msg = JSON.parse(event.data as string) as WSMessage;
        setLastMessage(msg);
        const handler = onEventRef.current?.[msg.type];
        if (handler) handler(msg);
      } catch {
        // Ignore non-JSON messages (pong, etc.)
      }
    };

    ws.onerror = (event: Event) => {
      onErrorRef.current?.(event);
    };

    ws.onclose = () => {
      if (!mountedRef.current) return;
      clearTimers();

      const delay = Math.min(
        BASE_RECONNECT_DELAY * Math.pow(2, reconnectAttemptRef.current),
        MAX_RECONNECT_DELAY,
      );
      reconnectAttemptRef.current += 1;
      updateState("reconnecting");

      reconnectTimerRef.current = setTimeout(() => {
        if (mountedRef.current) connectRef.current();
      }, delay);
    };
  };

  useEffect(() => {
    mountedRef.current = true;

    if (enabled && token) {
      connectRef.current();
    }

    return () => {
      mountedRef.current = false;
      clearTimers();
      if (wsRef.current) {
        wsRef.current.onclose = null; // Prevent reconnect on intentional close.
        wsRef.current.close();
        wsRef.current = null;
      }
      setState("disconnected");
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, enabled]);

  return { state, send, subscribe, unsubscribe, lastMessage };
}
