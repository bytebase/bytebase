import type { AgentMessage, Message } from "./types";

const ERROR_PREFIX = "Error: ";

const isHistoricalAssistantError = (message: AgentMessage): boolean => {
  if (message.role !== "assistant") {
    return false;
  }

  if (
    typeof message.metadata?.error === "string" &&
    message.metadata.error.trim()
  ) {
    return true;
  }

  return Boolean(message.content?.startsWith(ERROR_PREFIX));
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
  messages: AgentMessage[]
): Message[] => {
  return messages
    .filter((message) => !isHistoricalAssistantError(message))
    .map(cloneMessage);
};

export const buildOutboundHistory = (
  systemPrompt: string,
  messages: AgentMessage[]
): Message[] => {
  return [
    { role: "system", content: systemPrompt },
    ...sanitizeOutboundHistory(messages),
  ];
};
