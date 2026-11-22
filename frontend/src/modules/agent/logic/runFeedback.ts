import { getAgentRunContinuation, getAgentRunStartup } from "./runCheckpoint";
import { getRunTokenBudget } from "./tokenBudget";
import type { AgentChat, AgentMessage } from "./types";

type AgentRunFeedback =
  | "execution-unavailable"
  | "storage"
  | "ai-configuration"
  | "continuation-unavailable"
  | "token-budget"
  | "error";

// The transcript and composer share one current blocker; diagnostics stay on that notice.
export function getAgentRunFeedback({
  executionAvailable,
  storageBlocked = false,
  aiConfigurationRequired,
  recoverySupported,
  remoteBusy = false,
  continuationUnavailable = false,
  chat,
  messages,
  pendingAttemptId,
}: {
  executionAvailable: boolean;
  storageBlocked?: boolean;
  aiConfigurationRequired: boolean;
  recoverySupported: boolean;
  remoteBusy?: boolean;
  continuationUnavailable?: boolean;
  chat: AgentChat | null;
  messages: readonly AgentMessage[];
  pendingAttemptId?: string;
}): AgentRunFeedback | undefined {
  if (!executionAvailable) return "execution-unavailable";
  if (storageBlocked) return "storage";
  if (aiConfigurationRequired) return "ai-configuration";
  if (remoteBusy) return undefined;
  const hasRun = Boolean(chat?.runId || chat?.runCheckpoint);
  const hasStartup = Boolean(getAgentRunStartup(chat, messages));
  if (
    (continuationUnavailable &&
      !hasStartup &&
      (chat?.status !== "error" || hasRun)) ||
    (recoverySupported &&
      chat?.status === "error" &&
      hasRun &&
      !chat.archived &&
      !hasStartup &&
      !getAgentRunContinuation(chat, messages))
  )
    return "continuation-unavailable";

  const projection = chat
    ? getRunTokenBudget(chat, pendingAttemptId)
    : undefined;
  if (projection && chat?.runCheckpoint?.suspension === "token-limit")
    return "token-budget";
  if (chat?.lastError) return "error";
  if (
    projection &&
    (chat?.status === "running" || chat?.status === "awaiting_user") &&
    projection.state !== "normal"
  )
    return "token-budget";
  return undefined;
}
