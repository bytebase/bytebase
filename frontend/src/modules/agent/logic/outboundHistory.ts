import {
  formatOutboundResourceReference,
  normalizeAgentResourceReferences,
} from "./resourceMention";
import type { AgentMessage, Message } from "./types";

const isHistoricalAssistantError = (message: AgentMessage): boolean => {
  return (
    message.role === "assistant" &&
    typeof message.metadata?.error === "string" &&
    Boolean(message.metadata.error.trim())
  );
};

const interruptionNote = {
  user: "This request was stopped by the user before completion. In-flight actions may have completed; verify current state before repeating them.",
  unexpected:
    "This request was interrupted unexpectedly before completion. In-flight actions may have completed; verify current state before repeating them.",
} as const;

const interruptionStatus = (reason: "user" | "unexpected") =>
  `[Agent run status: ${interruptionNote[reason]}]`;

const cloneMessage = ({
  role,
  content,
  toolCallId,
  toolCalls,
}: Message): Message => ({
  role,
  content,
  toolCallId,
  toolCalls: toolCalls?.map((toolCall) => ({ ...toolCall })),
});

export const sanitizeOutboundHistory = (
  messages: AgentMessage[],
  chatContext?: string
): Message[] => {
  const currentUserMessage = messages.findLast(
    (message) => message.role === "user"
  );
  const currentReferences = normalizeAgentResourceReferences(
    currentUserMessage?.metadata?.resourceReferences,
    currentUserMessage?.metadata?.resourceReference
  );
  let omitTurn = false;
  let omitInterruptedTurnMessages = false;

  return messages.flatMap((message) => {
    const references =
      message.role === "user"
        ? normalizeAgentResourceReferences(
            message.metadata?.resourceReferences,
            message.metadata?.resourceReference
          )
        : [];
    if (message.role === "user") {
      omitInterruptedTurnMessages =
        message !== currentUserMessage &&
        (message.metadata?.interruptionReason === "user" ||
          message.metadata?.interruptionReason === "unexpected");
      omitTurn =
        Boolean(chatContext) &&
        currentReferences.length === 0 &&
        references.some(
          (reference) =>
            reference.kind === "database" ||
            reference.kind === "issue" ||
            reference.kind === "plan" ||
            reference.project !== chatContext
        );
    }
    if (omitTurn) {
      const reason = message.metadata?.interruptionReason;
      if (
        message.role === "user" &&
        omitInterruptedTurnMessages &&
        (reason === "user" || reason === "unexpected")
      ) {
        return [{ role: "user", content: interruptionStatus(reason) }];
      }
      return [];
    }
    if (
      (omitInterruptedTurnMessages &&
        (message.role === "assistant" || message.role === "tool")) ||
      isHistoricalAssistantError(message)
    ) {
      return [];
    }

    const outbound = cloneMessage(message);
    if (references.length > 0) {
      outbound.content = `${outbound.content ?? ""}${references.map(formatOutboundResourceReference).join("")}`;
    }
    if (omitInterruptedTurnMessages && message.role === "user") {
      const reason = message.metadata?.interruptionReason;
      if (reason === "user" || reason === "unexpected") {
        outbound.content = `${outbound.content ?? ""}\n\n${interruptionStatus(reason)}`;
      }
    }
    return [outbound];
  });
};

export const buildOutboundHistory = (
  systemPrompt: string,
  messages: AgentMessage[],
  chatContext?: string
): Message[] => {
  return [
    { role: "system", content: systemPrompt },
    ...sanitizeOutboundHistory(messages, chatContext),
  ];
};
