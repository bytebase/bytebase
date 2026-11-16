import {
  boundAgentQueryResult,
  DEFAULT_QUERY_PREVIEW_BYTES,
  getAgentQuerySampleInfo,
  omitAgentQueryRows,
} from "../logic/queryResultRetention";
import type { AgentMessage, ToolCall } from "../logic/types";
import type { PersistedAgentState } from "./agent";

const MAX_RETAINED_SAMPLE_BYTES = 512 * 1024;

const isQueryCall = (call: ToolCall): boolean => {
  if (call.name !== "call_api") return false;
  try {
    const args: unknown = JSON.parse(call.arguments);
    if (typeof args !== "object" || args === null || Array.isArray(args))
      return false;
    const operationId = (args as Record<string, unknown>).operationId;
    return (
      operationId === "SQLService/Query" ||
      operationId === "bytebase.v1.SQLService.Query"
    );
  } catch {
    return false;
  }
};

const omitMessageSample = (
  message: AgentMessage,
  content = omitAgentQueryRows(message.content!)
): AgentMessage => {
  if (
    content === message.content &&
    message.metadata?.queryResultRetention?.version === 1 &&
    message.metadata.queryResultRetention.level === "omitted"
  ) {
    return message;
  }
  return {
    ...message,
    content,
    metadata: {
      ...message.metadata,
      queryResultRetention: { version: 1, level: "omitted" },
    },
  };
};

export function retainAgentQuerySamples(
  state: PersistedAgentState,
  maxSampleBytes = MAX_RETAINED_SAMPLE_BYTES
): PersistedAgentState {
  let messagesByChatId = state.messagesByChatId;
  const candidates: {
    chatId: string;
    index: number;
    createdTs: number;
    bytes: number;
  }[] = [];
  let totalBytes = 0;

  const replaceMessage = (
    chatId: string,
    index: number,
    message: AgentMessage
  ) => {
    const messages = messagesByChatId[chatId];
    if (messages[index] === message) return;
    if (messagesByChatId === state.messagesByChatId) {
      messagesByChatId = { ...messagesByChatId };
    }
    if (messages === state.messagesByChatId[chatId]) {
      messagesByChatId[chatId] = [...messages];
    }
    messagesByChatId[chatId][index] = message;
  };

  for (const [chatId, messages] of Object.entries(state.messagesByChatId)) {
    // Later duplicate calls also make an earlier result unsafe to classify.
    const calls = new Map<string, { call: ToolCall; index: number } | null>();
    for (const [index, message] of messages.entries()) {
      if (message.role !== "assistant") continue;
      for (const call of message.toolCalls ?? []) {
        if (!call.id) continue;
        calls.set(call.id, calls.has(call.id) ? null : { call, index });
      }
    }
    for (const [index, message] of messages.entries()) {
      if (
        message.role !== "tool" ||
        !message.toolCallId ||
        message.metadata?.pending === true ||
        state.pendingAskByChatId[chatId]?.toolCallId === message.toolCallId ||
        typeof message.content !== "string"
      ) {
        continue;
      }
      const matched = calls.get(message.toolCallId);
      if (!matched || matched.index >= index || !isQueryCall(matched.call))
        continue;
      const marker = message.metadata?.queryResultRetention;
      if (
        marker !== undefined &&
        (marker === null || marker.version !== 1 || marker.level !== "omitted")
      ) {
        continue;
      }
      const info = getAgentQuerySampleInfo(message.content);
      if (!info) continue;
      if (marker || info.preview?.level === "omitted") {
        replaceMessage(
          chatId,
          index,
          omitMessageSample(
            message,
            info.preview?.level === "omitted"
              ? message.content
              : omitAgentQueryRows(message.content)
          )
        );
        continue;
      }

      let retained = message;
      let bytes = info.bytes;
      if (!info.preview || bytes > DEFAULT_QUERY_PREVIEW_BYTES) {
        const content = boundAgentQueryResult(message.content);
        if (content !== message.content) retained = { ...message, content };
        if (bytes > DEFAULT_QUERY_PREVIEW_BYTES) {
          bytes = getAgentQuerySampleInfo(content)!.bytes;
        }
      }
      replaceMessage(chatId, index, retained);
      if (bytes > 0) {
        candidates.push({ chatId, index, createdTs: message.createdTs, bytes });
        totalBytes += bytes;
      }
    }
  }

  candidates.sort((a, b) => a.createdTs - b.createdTs);
  for (const candidate of candidates) {
    if (totalBytes <= maxSampleBytes) break;
    const { chatId, index, bytes } = candidate;
    replaceMessage(
      chatId,
      index,
      omitMessageSample(messagesByChatId[chatId][index])
    );
    totalBytes -= bytes;
  }

  return messagesByChatId === state.messagesByChatId
    ? state
    : { ...state, messagesByChatId };
}

export function omitAgentQuerySamples(
  state: PersistedAgentState
): PersistedAgentState {
  return retainAgentQuerySamples(state, 0);
}
