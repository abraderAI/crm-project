"use client";

import dynamic from "next/dynamic";

/** Dynamically load ChatbotWidget without SSR — safe to use from Server Components. */
const ChatbotWidget = dynamic(
  () => import("@/components/chatbot-widget").then((mod) => mod.ChatbotWidget),
  { ssr: false },
);

/** Thin client wrapper so Server Components can mount the ChatbotWidget. */
export function ChatbotWidgetLoader(): React.ReactNode {
  return <ChatbotWidget />;
}
