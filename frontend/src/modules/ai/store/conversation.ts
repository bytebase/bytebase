import { v1 as uuidv1 } from "uuid";
import { create } from "zustand";
import { useAppStore } from "@/stores/app";
import type { SQLEditorConnection } from "@/types";
import { storageKeyAiConversations } from "@/utils/storage-keys";
import type { Conversation, Message } from "../types";
import {
  loadConversationHistory,
  mutateConversationHistory,
} from "./conversationStorage";

type ConversationCreate = Omit<
  Conversation,
  "id" | "created_ts" | "messageList"
>;
type MessageCreate = Omit<Message, "id" | "created_ts" | "conversation"> & {
  conversation_id: string;
};

const connectionKey = (conn: { instance: string; database: string }) =>
  `${conn.instance}/${conn.database}`;

const currentStorageKey = () => {
  const state = useAppStore.getState();
  const email = state.currentUser?.email;
  const workspace =
    state.currentUser?.workspace || state.serverInfo?.workspace || "";
  if (!email || !workspace) return undefined;
  return storageKeyAiConversations(workspace, email);
};

const withMessageBackReferences = (
  conversation: Omit<Conversation, "messageList">,
  messages: Message[]
): Conversation => {
  const next: Conversation = { ...conversation, messageList: [] };
  next.messageList = messages.map((message) => ({
    ...message,
    conversation: next,
  }));
  return next;
};

type ConversationState = {
  conversationsById: Record<string, Conversation>;
  readyByConnection: Record<string, boolean>;
  storageKey?: string;
  fetchConversationListByConnection: (
    conn: SQLEditorConnection
  ) => Promise<Conversation[]>;
  createConversation: (
    conversationCreate: ConversationCreate
  ) => Promise<Conversation>;
  updateConversation: (conversation: Conversation) => Promise<Conversation>;
  deleteConversation: (id: string) => Promise<void>;
  createMessage: (messageCreate: MessageCreate) => Promise<Message>;
  updateMessage: (message: Message) => Promise<Message>;
  reset: () => Promise<void>;
};

let persistenceWarningShown = false;

export const useConversationStore = create<ConversationState>((set, get) => {
  const persist = async (
    key: string | undefined,
    mutate: (conversations: Conversation[]) => Conversation[]
  ) => {
    if (!key || get().storageKey !== key || currentStorageKey() !== key) return;
    const saved = await mutateConversationHistory(key, (conversations) => {
      if (get().storageKey !== key || currentStorageKey() !== key) return;
      return mutate(conversations);
    });
    if (!saved && !persistenceWarningShown) {
      persistenceWarningShown = true;
      console.warn("[AI Assistant] Unable to persist conversation history");
    }
  };

  const fetchConversationListByConnection = async (
    conn: SQLEditorConnection
  ): Promise<Conversation[]> => {
    const key = currentStorageKey();
    const existing = get();
    const allConversations =
      existing.storageKey === key
        ? Object.values(existing.conversationsById)
        : key
          ? loadConversationHistory(key)
          : [];
    const conversationsById = Object.fromEntries(
      allConversations
        .filter((conversation) => conversation.messageList.length > 0)
        .map((conversation) => [conversation.id, conversation])
    );
    set({
      conversationsById,
      storageKey: key,
      readyByConnection: {
        ...(existing.storageKey === key ? existing.readyByConnection : {}),
        [connectionKey(conn)]: true,
      },
    });
    return Object.values(conversationsById)
      .filter(
        (conversation) =>
          conversation.instance === conn.instance &&
          conversation.database === conn.database
      )
      .sort((a, b) => a.created_ts - b.created_ts);
  };

  const createConversation = async (
    conversationCreate: ConversationCreate
  ): Promise<Conversation> => {
    const key = currentStorageKey();
    const conversation: Conversation = {
      id: uuidv1(),
      created_ts: Date.now(),
      messageList: [],
      ...conversationCreate,
    };
    set((state) => ({
      conversationsById: {
        ...(state.storageKey === key
          ? state.conversationsById
          : Object.fromEntries(
              (key ? loadConversationHistory(key) : []).map((stored) => [
                stored.id,
                stored,
              ])
            )),
        [conversation.id]: conversation,
      },
      readyByConnection:
        state.storageKey === key ? state.readyByConnection : {},
      storageKey: key,
    }));
    await persist(key, (conversations) => [
      ...conversations.filter(({ id }) => id !== conversation.id),
      conversation,
    ]);
    return conversation;
  };

  const updateConversation = async (
    conversation: Conversation
  ): Promise<Conversation> => {
    const key = currentStorageKey();
    const existing = get().conversationsById[conversation.id];
    const next = withMessageBackReferences(
      conversation,
      existing?.messageList ?? conversation.messageList
    );
    set((state) => ({
      conversationsById: {
        ...state.conversationsById,
        [next.id]: next,
      },
    }));
    await persist(key, (conversations) =>
      conversations.map((stored) =>
        stored.id === next.id
          ? withMessageBackReferences(
              {
                ...stored,
                name: next.name,
                instance: next.instance,
                database: next.database,
              },
              stored.messageList
            )
          : stored
      )
    );
    return next;
  };

  const deleteConversation = async (id: string): Promise<void> => {
    const key = currentStorageKey();
    set((state) => {
      const conversationsById = { ...state.conversationsById };
      delete conversationsById[id];
      return { conversationsById };
    });
    await persist(key, (conversations) =>
      conversations.filter((conversation) => conversation.id !== id)
    );
  };

  const createMessage = async (
    messageCreate: MessageCreate
  ): Promise<Message> => {
    const key = currentStorageKey();
    const conversation = get().conversationsById[messageCreate.conversation_id];
    if (!conversation) {
      throw new Error(
        `Conversation not found: ${messageCreate.conversation_id}`
      );
    }
    const message = {
      id: uuidv1(),
      created_ts: Date.now(),
      author: messageCreate.author,
      content: messageCreate.content,
      status: messageCreate.status,
      error: messageCreate.error,
      conversation,
    } satisfies Message;
    const next = withMessageBackReferences(conversation, [
      ...conversation.messageList,
      message,
    ]);
    set((state) => ({
      conversationsById: {
        ...state.conversationsById,
        [next.id]: next,
      },
    }));
    await persist(key, (conversations) => {
      const stored = conversations.find(({ id }) => id === next.id);
      if (!stored) return [...conversations, next];
      const messages = stored.messageList.some(({ id }) => id === message.id)
        ? stored.messageList
        : [...stored.messageList, message].sort(
            (a, b) => a.created_ts - b.created_ts
          );
      const merged = withMessageBackReferences(stored, messages);
      return conversations.map((conversation) =>
        conversation.id === merged.id ? merged : conversation
      );
    });
    return next.messageList.at(-1)!;
  };

  const updateMessage = async (message: Message): Promise<Message> => {
    const key = currentStorageKey();
    const conversation = get().conversationsById[message.conversation.id];
    if (!conversation) return message;
    const next = withMessageBackReferences(
      conversation,
      conversation.messageList.map((current) =>
        current.id === message.id ? message : current
      )
    );
    set((state) => ({
      conversationsById: {
        ...state.conversationsById,
        [next.id]: next,
      },
    }));
    await persist(key, (conversations) => {
      const stored = conversations.find(({ id }) => id === next.id);
      if (!stored) return conversations;
      const exists = stored.messageList.some(({ id }) => id === message.id);
      const messages = exists
        ? stored.messageList.map((current) =>
            current.id === message.id ? message : current
          )
        : [...stored.messageList, message].sort(
            (a, b) => a.created_ts - b.created_ts
          );
      const merged = withMessageBackReferences(stored, messages);
      return conversations.map((conversation) =>
        conversation.id === merged.id ? merged : conversation
      );
    });
    return next.messageList.find(({ id }) => id === message.id) ?? message;
  };

  const reset = async () => {
    set({
      conversationsById: {},
      readyByConnection: {},
      storageKey: undefined,
    });
  };

  return {
    conversationsById: {},
    readyByConnection: {},
    storageKey: undefined,
    fetchConversationListByConnection,
    createConversation,
    updateConversation,
    deleteConversation,
    createMessage,
    updateMessage,
    reset,
  };
});

export const conversationListByConnection = (
  state: ConversationState,
  conn: { instance: string; database: string }
): Conversation[] =>
  Object.values(state.conversationsById)
    .filter((c) => c.instance === conn.instance && c.database === conn.database)
    .sort((a, b) => a.created_ts - b.created_ts);

export const isConnectionReady = (
  state: ConversationState,
  conn: { instance: string; database: string }
): boolean => !!state.readyByConnection[connectionKey(conn)];
