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
