import type { PersistedAgentState } from "./agent";

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
        !chat.runCheckpoint &&
        !state.pendingAskByChatId[chat.id] &&
        !state.queuedUserMessagesByChatId[chat.id] &&
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
  chatIds: ReadonlySet<string>
): PersistedAgentState {
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
    ...(activeChatIds && {
      activeChatIds:
        activeChatIds.length === state.activeChatIds?.length
          ? state.activeChatIds
          : activeChatIds,
    }),
    currentChatId,
  };
}
