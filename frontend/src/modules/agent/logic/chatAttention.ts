import { getAgentRunContinuation } from "./runCheckpoint";
import type { AgentChat, AgentMessage, AgentPendingAsk } from "./types";

export function chatNeedsAttention(
  chat: AgentChat | null | undefined,
  messages: readonly AgentMessage[],
  pendingAsk: AgentPendingAsk | null | undefined
): boolean {
  if (!chat || chat.archived) return false;
  if (chat.status === "awaiting_user" && pendingAsk) return true;
  return (
    chat.runCheckpoint?.suspension === "token-limit" &&
    getAgentRunContinuation(chat, messages) !== undefined
  );
}
