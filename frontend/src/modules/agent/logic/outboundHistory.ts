import {
  formatOutboundResourceReference,
  normalizeAgentResourceReference,
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
  const currentReference = normalizeAgentResourceReference(
    currentUserMessage?.metadata?.resourceReference
  );
  let omitTurn = false;

  return messages.flatMap((message) => {
    const reference =
      message.role === "user"
        ? normalizeAgentResourceReference(message.metadata?.resourceReference)
        : undefined;
    if (message.role === "user") {
      omitTurn =
        Boolean(chatContext) &&
        !currentReference &&
        reference?.kind === "project" &&
        reference.project !== chatContext;
    }
    if (omitTurn || isHistoricalAssistantError(message)) return [];

    const outbound = cloneMessage(message);
    if (reference) {
      outbound.content = `${outbound.content ?? ""}${formatOutboundResourceReference(reference)}`;
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
