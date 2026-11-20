import {
  normalizeAgentApiAttempts,
  reconcileApiResultLinks,
} from "../logic/apiAttempt";
import type { AgentChat, AgentMessage } from "../logic/types";
import type { PersistedAgentState } from "./agent";

export const hasUnreadableApiHistory = (
  chat: Pick<AgentChat, "id"> & {
    apiAttempts?: unknown;
  },
  messages: readonly AgentMessage[],
  scopeKey?: string
) => {
  const attempts = normalizeAgentApiAttempts(chat.apiAttempts, chat.id);
  const reconciled = reconcileApiResultLinks(
    chat.id,
    scopeKey,
    attempts,
    messages.map((message) => ({
      ...message,
      chatId: message.chatId ?? chat.id,
    }))
  );
  return !!reconciled?.some(
    (attempt) => attempt.execution?.reason === "journal-unreadable"
  );
};

export const protectsAgentChatFromDeletion = (
  state: Pick<PersistedAgentState, "messagesByChatId" | "pendingAskByChatId">,
  chat: AgentChat,
  scopeKey?: string
) =>
  hasUnreadableApiHistory(
    chat,
    state.messagesByChatId[chat.id] ?? [],
    scopeKey
  );

export function getAgentHistoryEvictionCandidates(
  state: PersistedAgentState,
  protectedChatIds: ReadonlySet<string>
): string[] {
  const activeChatIds = new Set(state.activeChatIds);
  return state.chats
    .filter((chat) => {
      const draft = state.draftsByChatId[chat.id];
      return (
        (chat.status === "idle" || chat.status === "error") &&
        chat.id !== state.currentChatId &&
        !protectedChatIds.has(chat.id) &&
        !activeChatIds.has(chat.id) &&
        !chat.runId &&
        !protectsAgentChatFromDeletion(state, chat) &&
        !chat.runCheckpoint &&
        !state.pendingAskByChatId[chat.id] &&
        !state.queuedUserMessagesByChatId[chat.id] &&
        !state.followUpQueuesByChatId?.[chat.id]?.messages.length &&
        !(
          draft?.text ||
          draft?.project ||
          draft?.resource ||
          draft?.mentions?.length
        ) &&
        !state.messagesByChatId[chat.id]?.some(
          (message) => message.metadata?.pending === true
        )
      );
    })
    .sort(
      (a, b) =>
        a.updatedTs - b.updatedTs ||
        a.createdTs - b.createdTs ||
        (a.id < b.id ? -1 : a.id > b.id ? 1 : 0)
    )
    .map((chat) => chat.id);
}

export function removeAgentHistoryChats(
  state: PersistedAgentState,
  chatIds: ReadonlySet<string>,
  scopeKey?: string
): PersistedAgentState {
  if (chatIds.size) {
    const removable = new Set(chatIds);
    for (const chat of state.chats)
      if (
        chatIds.has(chat.id) &&
        protectsAgentChatFromDeletion(state, chat, scopeKey)
      )
        removable.delete(chat.id);
    chatIds = removable;
  }
  if (!chatIds.size) return state;
  const withoutChats = <T>(records: Record<string, T>): Record<string, T> => {
    let retained = records;
    for (const id of chatIds) {
      if (!Object.hasOwn(records, id)) continue;
      if (retained === records) retained = { ...records };
      delete retained[id];
    }
    return retained;
  };
  const chats = state.chats.filter((chat) => !chatIds.has(chat.id));
  const messagesByChatId = withoutChats(state.messagesByChatId);
  const draftsByChatId = withoutChats(state.draftsByChatId);
  const pendingAskByChatId = withoutChats(state.pendingAskByChatId);
  const queuedUserMessagesByChatId = withoutChats(
    state.queuedUserMessagesByChatId
  );
  const followUpQueuesByChatId =
    state.followUpQueuesByChatId && withoutChats(state.followUpQueuesByChatId);
  const activeChatIds = state.activeChatIds?.filter((id) => !chatIds.has(id));
  const currentChatId =
    state.currentChatId && chatIds.has(state.currentChatId)
      ? null
      : state.currentChatId;
  if (
    chats.length === state.chats.length &&
    messagesByChatId === state.messagesByChatId &&
    draftsByChatId === state.draftsByChatId &&
    pendingAskByChatId === state.pendingAskByChatId &&
    queuedUserMessagesByChatId === state.queuedUserMessagesByChatId &&
    followUpQueuesByChatId === state.followUpQueuesByChatId &&
    activeChatIds?.length === state.activeChatIds?.length &&
    currentChatId === state.currentChatId
  ) {
    return state;
  }
  return {
    ...state,
    chats: chats.length === state.chats.length ? state.chats : chats,
    messagesByChatId,
    draftsByChatId,
    pendingAskByChatId,
    queuedUserMessagesByChatId,
    followUpQueuesByChatId:
      followUpQueuesByChatId && Object.keys(followUpQueuesByChatId).length
        ? followUpQueuesByChatId
        : undefined,
    ...(activeChatIds && {
      activeChatIds:
        activeChatIds.length === state.activeChatIds?.length
          ? state.activeChatIds
          : activeChatIds,
    }),
    currentChatId,
  };
}
