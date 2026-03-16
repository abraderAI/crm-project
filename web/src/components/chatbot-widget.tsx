"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useAuth } from "@clerk/nextjs";
import { buildHeaders, buildUrl } from "@/lib/api-client";

/** A single message in the chatbot conversation. */
interface ChatMessage {
  id: string;
  role: "user" | "assistant";
  content: string;
}

/** Response shape from POST /v1/chat/message. */
interface ChatResponse {
  reply: string;
}

/**
 * In-app chatbot widget with floating bubble and expandable chat panel.
 * Renders on all pages (authenticated and public) via AppLayoutWrapper.
 * Sends messages to POST /v1/chat/message with optional Clerk auth token.
 */
export function ChatbotWidget(): React.ReactNode {
  const [isOpen, setIsOpen] = useState(false);
  const [input, setInput] = useState("");
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { getToken } = useAuth();
  const tokenRef = useRef<string | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Resolve Clerk token on mount and when auth state changes.
  useEffect(() => {
    let active = true;
    getToken().then((t) => {
      if (active) tokenRef.current = t;
    });
    return () => {
      active = false;
    };
  }, [getToken]);

  // Auto-scroll to bottom when messages change.
  useEffect(() => {
    if (typeof messagesEndRef.current?.scrollIntoView === "function") {
      messagesEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages]);

  const sendMessage = useCallback(async () => {
    const trimmed = input.trim();
    if (!trimmed || isLoading) return;

    setError(null);
    setInput("");

    const userMessage: ChatMessage = {
      id: `user-${Date.now()}`,
      role: "user",
      content: trimmed,
    };
    setMessages((prev) => [...prev, userMessage]);
    setIsLoading(true);

    try {
      const url = buildUrl("/chat/message");
      const headers = buildHeaders(tokenRef.current);
      const response = await fetch(url, {
        method: "POST",
        headers,
        body: JSON.stringify({ message: trimmed }),
      });

      if (!response.ok) {
        setError("Failed to send message. Please try again.");
        return;
      }

      const data = (await response.json()) as ChatResponse;
      const botMessage: ChatMessage = {
        id: `bot-${Date.now()}`,
        role: "assistant",
        content: data.reply,
      };
      setMessages((prev) => [...prev, botMessage]);
    } catch {
      setError("Failed to send message. Please try again.");
    } finally {
      setIsLoading(false);
    }
  }, [input, isLoading]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        void sendMessage();
      }
    },
    [sendMessage],
  );

  return (
    <>
      {/* Floating chat bubble */}
      <button
        data-testid="chatbot-bubble"
        onClick={() => setIsOpen(!isOpen)}
        className="fixed bottom-6 right-6 z-50 flex h-14 w-14 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg transition-transform hover:scale-105"
        aria-label="Open chat"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
        </svg>
      </button>

      {/* Expandable chat panel */}
      {isOpen && (
        <div
          data-testid="chatbot-panel"
          className="fixed bottom-24 right-6 z-50 flex h-[28rem] w-80 flex-col rounded-lg border border-border bg-background shadow-xl"
        >
          {/* Header */}
          <div
            data-testid="chatbot-header"
            className="flex items-center justify-between rounded-t-lg border-b border-border bg-primary px-4 py-3 text-primary-foreground"
          >
            <span className="text-sm font-semibold">DEFT Assistant</span>
            <button
              data-testid="chatbot-close"
              onClick={() => setIsOpen(false)}
              className="text-primary-foreground/80 hover:text-primary-foreground"
              aria-label="Close chat"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </div>

          {/* Messages area */}
          <div className="flex-1 overflow-y-auto p-3">
            {messages.length === 0 && (
              <p className="text-center text-xs text-muted-foreground">
                Send a message to start chatting.
              </p>
            )}
            {messages.map((msg) => (
              <div
                key={msg.id}
                className={`mb-2 max-w-[85%] rounded-lg px-3 py-2 text-sm ${
                  msg.role === "user"
                    ? "ml-auto bg-primary text-primary-foreground"
                    : "mr-auto bg-muted text-foreground"
                }`}
              >
                {msg.content}
              </div>
            ))}
            {isLoading && (
              <div className="mr-auto mb-2 max-w-[85%] rounded-lg bg-muted px-3 py-2 text-sm text-muted-foreground">
                Typing…
              </div>
            )}
            {error && (
              <div
                data-testid="chatbot-error"
                className="mr-auto mb-2 max-w-[85%] rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive"
              >
                {error}
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>

          {/* Input area */}
          <div className="flex gap-2 border-t border-border p-3">
            <input
              data-testid="chatbot-input"
              type="text"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Type a message…"
              className="flex-1 rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            />
            <button
              data-testid="chatbot-send"
              onClick={() => void sendMessage()}
              disabled={isLoading}
              className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
              aria-label="Send message"
            >
              Send
            </button>
          </div>
        </div>
      )}
    </>
  );
}
