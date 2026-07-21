import { createContext, useContext } from "react";

export type Mode = "CHAT" | "VIEW";

export type ChatViewContext = {
  mode: Mode;
};

const ChatViewReactContext = createContext<ChatViewContext | null>(null);

export const ChatViewProvider = ChatViewReactContext.Provider;

export function useChatViewContext(): ChatViewContext {
  const ctx = useContext(ChatViewReactContext);
  if (!ctx) {
    throw new Error(
      "useChatViewContext must be used inside <ChatViewProvider>"
    );
  }
  return ctx;
}
