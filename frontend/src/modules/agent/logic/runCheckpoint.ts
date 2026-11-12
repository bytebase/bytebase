import { normalizeAgentDiagnosticCapability } from "./diagnosticCapability";
import { normalizeRunBudget, normalizeRunCheckpoint } from "./tokenBudget";
import type { AgentChat, AgentMessage, AgentRunCheckpoint } from "./types";

export interface AgentRunContinuation {
  checkpoint: AgentRunCheckpoint;
  messages: AgentMessage[];
}

export function getAgentRunContinuation(
  chat: AgentChat | null | undefined,
  messages: readonly AgentMessage[]
): AgentRunContinuation | undefined {
  if (!chat || chat.archived) return undefined;
  const budget = normalizeRunBudget(chat.runBudget);
  const checkpoint = normalizeRunCheckpoint(chat.runCheckpoint);
  if (
    !budget ||
    !checkpoint ||
    checkpoint.startupComplete !== true ||
    chat.runId !== checkpoint.runId ||
    budget.runId !== checkpoint.runId ||
    (checkpoint.phase === "finalizing" &&
      typeof checkpoint.finalizationText !== "string")
  )
    return undefined;

  const capability = normalizeAgentDiagnosticCapability(chat.capability);
  if (
    (chat.capability !== undefined && !capability) ||
    JSON.stringify(capability) !==
      JSON.stringify(checkpoint.bindings.capability)
  )
    return undefined;

  const ids = new Set<string>();
  let root = -1;
  let through = -1;
  for (const [index, message] of messages.entries()) {
    if (!message.id || ids.has(message.id) || message.chatId !== chat.id)
      return undefined;
    ids.add(message.id);
    if (message.id === checkpoint.rootUserMessageId) {
      if (
        message.role !== "user" ||
        (message.metadata?.runId !== undefined &&
          message.metadata.runId !== checkpoint.runId)
      )
        return undefined;
      root = index;
    }
    if (message.id === checkpoint.throughMessageId) {
      if (
        message.metadata?.runId !== undefined &&
        message.metadata.runId !== checkpoint.runId
      )
        return undefined;
      through = index;
    }
  }
  if (root < 0 || through < root) return undefined;

  // The cursor is a recovery boundary, not a transcript truncation point.
  // Later canonical records retain known outcomes for outbound repair without replay.
  return { checkpoint, messages: [...messages] };
}
