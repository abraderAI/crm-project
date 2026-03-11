"use client";

import { useCallback, useEffect, useState } from "react";
import type { Message, TypingPayload, WSMessage } from "@/lib/api-types";
import type { TypingUser } from "@/hooks/use-typing";
import { useTyping } from "@/hooks/use-typing";
import { MessageTimeline } from "@/components/thread/message-timeline";
import { TypingIndicator } from "./typing-indicator";

export interface RealtimeMessagesProps {
  /** Thread ID for channel subscription. */
  threadId: string;
  /** Initial messages loaded from the server. */
  initialMessages: Message[];
  /** Current user's ID. */
  currentUserId?: string;
  /** Called when the edit button on a message is clicked. */
  onEditMessage?: (messageId: string) => void;
  /** WebSocket subscribe function. */
  wsSubscribe?: (channel: string) => void;
  /** WebSocket unsubscribe function. */
  wsUnsubscribe?: (channel: string) => void;
  /** Send a typing event via WS. */
  wsSendTyping?: (threadId: string) => void;
  /** External typing users (from parent WS handler). */
  externalTypingUsers?: TypingUser[];
  /** External new messages from WS (from parent handler). */
  externalNewMessages?: Message[];
}

/** Thread message view with real-time updates and typing indicators. */
export function RealtimeMessages({
  threadId,
  initialMessages,
  currentUserId,
  onEditMessage,
  wsSubscribe,
  wsUnsubscribe,
  wsSendTyping,
  externalTypingUsers,
  externalNewMessages,
}: RealtimeMessagesProps): React.ReactNode {
  const [messages, setMessages] = useState<Message[]>(initialMessages);

  const { typingUsers, handleLocalTyping, handleRemoteTyping } = useTyping({
    threadId,
    currentUserId,
    sendTyping: wsSendTyping,
  });

  // Subscribe to the thread channel.
  useEffect(() => {
    const channel = `thread:${threadId}`;
    wsSubscribe?.(channel);
    return () => {
      wsUnsubscribe?.(channel);
    };
  }, [threadId, wsSubscribe, wsUnsubscribe]);

  // Sync initial messages when they change (e.g. re-fetch).
  useEffect(() => {
    setMessages(initialMessages);
  }, [initialMessages]);

  // Append external new messages from WS.
  useEffect(() => {
    if (!externalNewMessages || externalNewMessages.length === 0) return;
    setMessages((prev) => {
      const existingIds = new Set(prev.map((m) => m.id));
      const newMsgs = externalNewMessages.filter((m) => !existingIds.has(m.id));
      if (newMsgs.length === 0) return prev;
      return [...prev, ...newMsgs];
    });
  }, [externalNewMessages]);

  /** Add a new message from a WS event. */
  const addMessage = useCallback((msg: Message) => {
    setMessages((prev) => {
      if (prev.some((m) => m.id === msg.id)) return prev;
      return [...prev, msg];
    });
  }, []);

  /** Update an existing message from a WS event. */
  const updateMessage = useCallback((msg: Message) => {
    setMessages((prev) => prev.map((m) => (m.id === msg.id ? msg : m)));
  }, []);

  /** Process a WS message event. */
  const handleWSMessage = useCallback(
    (wsMsg: WSMessage) => {
      if (wsMsg.type === "message.created") {
        addMessage(wsMsg.payload as Message);
      } else if (wsMsg.type === "message.updated") {
        updateMessage(wsMsg.payload as Message);
      } else if (wsMsg.type === "typing") {
        handleRemoteTyping(wsMsg as WSMessage<TypingPayload>);
      }
    },
    [addMessage, updateMessage, handleRemoteTyping],
  );

  // Merge external typing users with local tracking.
  const allTypingUsers = externalTypingUsers ?? typingUsers;

  return (
    <div data-testid="realtime-messages">
      <MessageTimeline messages={messages} currentUserId={currentUserId} onEdit={onEditMessage} />
      <TypingIndicator typingUsers={allTypingUsers} />
      {/* Expose handleWSMessage and handleLocalTyping for parent integration. */}
      <input type="hidden" data-handler-ws={String(!!handleWSMessage)} />
      <input type="hidden" data-handler-typing={String(!!handleLocalTyping)} />
    </div>
  );
}
