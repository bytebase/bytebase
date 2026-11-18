import type { PersistedAgentState } from "./agent";
import { omitAgentQuerySamples } from "./storageRetention";

type RecordValue = Record<string, unknown>;

interface StoredState extends RecordValue {
  chats: RecordValue[];
  messagesByChatId: Record<string, RecordValue[]>;
}

const envelopeKeys = new Set([
  "currentChatId",
  "chats",
  "messagesByChatId",
  "pendingAskByChatId",
  "draftsByChatId",
  "queuedUserMessagesByChatId",
  "followUpQueuesByChatId",
  "activeChatIds",
]);
const transientChatKeys = new Set([
  "status",
  "lastError",
  "requiresAIConfiguration",
  "interrupted",
  "interruptionReason",
  "archived",
]);
const statuses = new Set([
  "idle",
  "error",
  "running",
  "paused",
  "awaiting_user",
]);
const roles = new Set(["system", "user", "assistant", "tool"]);

const isRecord = (value: unknown): value is RecordValue =>
  typeof value === "object" &&
  value !== null &&
  !Array.isArray(value) &&
  (Object.getPrototypeOf(value) === Object.prototype ||
    Object.getPrototypeOf(value) === null);

const isJson = (value: unknown): boolean => {
  if (value === null || typeof value === "string" || typeof value === "boolean")
    return true;
  if (typeof value === "number") return Number.isFinite(value);
  if (Array.isArray(value)) return value.every(isJson);
  return isRecord(value) && Object.values(value).every(isJson);
};

const isIdentity = (value: unknown): value is string =>
  typeof value === "string" && value.length > 0;

const isFiniteNonnegative = (value: unknown): value is number =>
  typeof value === "number" && Number.isFinite(value) && value >= 0;

const isStoredState = (value: unknown): value is StoredState => {
  if (
    !isRecord(value) ||
    !isJson(value) ||
    !Array.isArray(value.chats) ||
    !isRecord(value.messagesByChatId)
  )
    return false;
  const chatIds = new Set<string>();
  for (const chat of value.chats) {
    if (
      !isRecord(chat) ||
      !isIdentity(chat.id) ||
      chatIds.has(chat.id) ||
      (chat.status !== undefined && !statuses.has(String(chat.status))) ||
      (chat.archived !== undefined && typeof chat.archived !== "boolean") ||
      (chat.lastError !== undefined &&
        chat.lastError !== null &&
        typeof chat.lastError !== "string") ||
      (chat.requiresAIConfiguration !== undefined &&
        typeof chat.requiresAIConfiguration !== "boolean") ||
      (chat.interrupted !== undefined &&
        typeof chat.interrupted !== "boolean") ||
      (chat.hasUnreportedUsage !== undefined &&
        typeof chat.hasUnreportedUsage !== "boolean") ||
      (chat.interruptionReason !== undefined &&
        chat.interruptionReason !== "user" &&
        chat.interruptionReason !== "unexpected") ||
      (chat.runId !== undefined &&
        chat.runId !== null &&
        typeof chat.runId !== "string") ||
      (chat.runCheckpoint !== undefined &&
        chat.runCheckpoint !== null &&
        !isRecord(chat.runCheckpoint)) ||
      (chat.updatedTs !== undefined && !isFiniteNonnegative(chat.updatedTs)) ||
      (chat.totalTokensUsed !== undefined &&
        !isFiniteNonnegative(chat.totalTokensUsed))
    )
      return false;
    chatIds.add(chat.id);
  }
  if (
    value.currentChatId !== undefined &&
    value.currentChatId !== null &&
    (typeof value.currentChatId !== "string" ||
      !chatIds.has(value.currentChatId))
  )
    return false;
  if (
    value.activeChatIds !== undefined &&
    (!Array.isArray(value.activeChatIds) ||
      !value.activeChatIds.every(
        (id) => typeof id === "string" && chatIds.has(id)
      ))
  )
    return false;
  for (const key of [
    "draftsByChatId",
    "pendingAskByChatId",
    "queuedUserMessagesByChatId",
    "followUpQueuesByChatId",
  ]) {
    const records = value[key];
    if (records === undefined) continue;
    if (
      !isRecord(records) ||
      Object.entries(records).some(
        ([id, record]) => !chatIds.has(id) || !isRecord(record)
      )
    )
      return false;
  }
  for (const [chatId, messages] of Object.entries(value.messagesByChatId)) {
    if (!chatIds.has(chatId) || !Array.isArray(messages)) return false;
    const messageIds = new Set<string>();
    for (const message of messages) {
      if (
        !isRecord(message) ||
        !isIdentity(message.id) ||
        messageIds.has(message.id) ||
        !roles.has(String(message.role)) ||
        (message.chatId !== undefined && message.chatId !== chatId) ||
        (message.content !== undefined &&
          message.content !== null &&
          typeof message.content !== "string") ||
        (message.metadata !== undefined && !isRecord(message.metadata))
      )
        return false;
      messageIds.add(message.id);
    }
  }
  return true;
};

const containsValue = (source: unknown, saved: unknown): boolean => {
  if (Array.isArray(source))
    return (
      Array.isArray(saved) &&
      source.length === saved.length &&
      source.every((value, index) => containsValue(value, saved[index]))
    );
  if (isRecord(source))
    return (
      isRecord(saved) &&
      Object.entries(source).every(
        ([key, value]) =>
          Object.hasOwn(saved, key) && containsValue(value, saved[key])
      )
    );
  return source === saved;
};

const hasOnlyKeys = (value: unknown, keys: readonly string[]): boolean =>
  isRecord(value) && Object.keys(value).every((key) => keys.includes(key));

const hasRecognizedSampleMarkers = (message: RecordValue): boolean => {
  if (
    isRecord(message.metadata) &&
    Object.hasOwn(message.metadata, "queryResultRetention") &&
    !hasOnlyKeys(message.metadata.queryResultRetention, ["version", "level"])
  )
    return false;
  if (typeof message.content !== "string") return false;
  try {
    const content: unknown = JSON.parse(message.content);
    if (!isRecord(content)) return false;
    if (!Object.hasOwn(content, "agentQueryPreview")) return true;
    const preview = content.agentQueryPreview;
    return (
      hasOnlyKeys(preview, ["version", "level", "results"]) &&
      isRecord(preview) &&
      Array.isArray(preview.results) &&
      preview.results.every((result) =>
        hasOnlyKeys(result, ["receivedRows", "retainedRows"])
      )
    );
  } catch {
    return false;
  }
};

const omitQuerySamples = (state: StoredState): StoredState =>
  omitAgentQuerySamples({
    ...state,
    pendingAskByChatId: state.pendingAskByChatId ?? {},
  } as unknown as PersistedAgentState) as unknown as StoredState;

// Compare decoded raw states before normalization can hide activity or unknown data.
export function canRetireLegacyAgentState(
  source: unknown,
  saved: unknown
): boolean {
  if (
    !isStoredState(source) ||
    !isStoredState(saved) ||
    Object.keys(source).some((key) => !envelopeKeys.has(key)) ||
    (Array.isArray(source.activeChatIds) && source.activeChatIds.length > 0) ||
    Object.keys(source.pendingAskByChatId ?? {}).length > 0 ||
    Object.keys(source.queuedUserMessagesByChatId ?? {}).length > 0 ||
    Object.keys(source.followUpQueuesByChatId ?? {}).length > 0 ||
    Object.values(source.messagesByChatId).some((messages) =>
      messages.some(
        (message) => isRecord(message.metadata) && message.metadata.pending
      )
    )
  )
    return false;
  const retainedChats = new Map(saved.chats.map((chat) => [chat.id, chat]));
  for (const chat of source.chats) {
    if (
      (chat.status !== undefined &&
        chat.status !== "idle" &&
        chat.status !== "error") ||
      chat.runId ||
      chat.runCheckpoint
    )
      return false;
    const retained = retainedChats.get(chat.id);
    if (
      !retained ||
      !Object.entries(chat).every(([key, value]) => {
        if (transientChatKeys.has(key)) return true;
        if (key === "updatedTs" || key === "totalTokensUsed")
          return (
            isFiniteNonnegative(retained[key]) &&
            retained[key] >= (value as number)
          );
        if (key === "hasUnreportedUsage")
          return (
            typeof retained[key] === "boolean" &&
            (!value || retained[key] === true)
          );
        return (
          Object.hasOwn(retained, key) && containsValue(value, retained[key])
        );
      })
    )
      return false;
  }
  let omittedSamples: { source: StoredState; saved: StoredState } | undefined;
  for (const [chatId, messages] of Object.entries(source.messagesByChatId)) {
    const retained = saved.messagesByChatId[chatId];
    if (!retained) return false;
    let cursor = 0;
    for (const [index, message] of messages.entries()) {
      while (cursor < retained.length && retained[cursor].id !== message.id)
        cursor++;
      if (cursor === retained.length) return false;
      if (!containsValue(message, retained[cursor])) {
        if (!hasRecognizedSampleMarkers(message)) return false;
        try {
          omittedSamples ??= {
            source: omitQuerySamples(source),
            saved: omitQuerySamples(saved),
          };
        } catch {
          return false;
        }
        if (
          !containsValue(
            omittedSamples.source.messagesByChatId[chatId][index],
            omittedSamples.saved.messagesByChatId[chatId][cursor]
          )
        )
          return false;
      }
      cursor++;
    }
  }
  return containsValue(source.draftsByChatId ?? {}, saved.draftsByChatId ?? {});
}
