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

  return messages.flatMap((message) => {
    const references =
      message.role === "user"
        ? normalizeAgentResourceReferences(
            message.metadata?.resourceReferences,
            message.metadata?.resourceReference
          )
        : [];
    if (message.role === "user") {
      omitTurn =
        Boolean(chatContext) &&
        currentReferences.length === 0 &&
        references.some(
          (reference) =>
            reference.kind === "database" || reference.project !== chatContext
        );
    }
    if (omitTurn || isHistoricalAssistantError(message)) return [];

    const outbound = cloneMessage(message);
    if (references.length > 0) {
      outbound.content = `${outbound.content ?? ""}${references.map(formatOutboundResourceReference).join("")}`;
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
