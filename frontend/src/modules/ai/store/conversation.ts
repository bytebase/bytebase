import { groupBy, omit } from "lodash-es";
import PouchDB from "pouchdb";
import PouchDBFind from "pouchdb-find";
import { v1 as uuidv1 } from "uuid";
import { create } from "zustand";
import type { SQLEditorConnection } from "@/types";
import type { Conversation, Message } from "../types";

type RowStatus = "NORMAL" | "ARCHIVED";

type Entity<T, O extends keyof T, E = unknown> = Omit<T, "id" | O> & {
  _id: string;
  _rev?: string;
  row_status: RowStatus;
} & E;

type ConversationEntity = Entity<Conversation, "messageList">;
type MessageEntity = Entity<
  Message,
  "conversation",
  {
    conversation_id: string;
  }
>;

type EntityCreate<T> = Omit<T, "_id" | "created_ts" | "row_status">;

PouchDB.plugin(PouchDBFind);

const convertConversationToEntity = (
  conversation: Conversation
): ConversationEntity => ({
  ...omit(conversation, "id", "messageList"),
  _id: conversation.id,
  row_status: "NORMAL",
});

const convertMessageToEntity = (message: Message): MessageEntity => ({
  ...omit(message, "id", "conversation"),
  _id: message.id,
  row_status: "NORMAL",
  conversation_id: message.conversation.id,
});

const FK_MESSAGE_CONVERSATION_ID = "fk_message_conversation_id";

const conversations = new PouchDB<ConversationEntity>(
  "bb.plugin.ai.conversations"
);
const messages = new PouchDB<MessageEntity>("bb.plugin.ai.messages");
const ready: Promise<unknown>[] = [];
ready.push(
  messages.createIndex({
    index: { name: FK_MESSAGE_CONVERSATION_ID, fields: ["conversation_id"] },
  })
);

const connectionKey = (conn: { instance: string; database: string }) =>
  `${conn.instance}/${conn.database}`;

type ConversationState = {
  conversationsById: Record<string, Conversation>;
  readyByConnection: Record<string, boolean>;
  fetchConversationListByConnection: (
    conn: SQLEditorConnection
  ) => Promise<Conversation[]>;
  createConversation: (
    conversationCreate: EntityCreate<ConversationEntity>
  ) => Promise<Conversation>;
  updateConversation: (conversation: Conversation) => Promise<Conversation>;
  deleteConversation: (id: string) => Promise<void>;
  createMessage: (
    messageCreate: EntityCreate<MessageEntity>
  ) => Promise<Message>;
  updateMessage: (message: Message) => Promise<Message>;
  reset: () => Promise<void>;
};

// Build a plain Conversation (with its message back-refs) from entities.
const buildConversation = (
  c: ConversationEntity,
  messageEntities: MessageEntity[]
): Conversation => {
  const conversation: Conversation = {
    ...omit(c, "_id", "_rev", "row_status"),
    id: c._id,
    messageList: [],
  } as Conversation;
  conversation.messageList = messageEntities
    .map<Message>(
      (m) =>
        ({
          ...omit(m, "_id", "_rev", "row_status", "conversation_id"),
          id: m._id,
          conversation,
        }) as Message
    )
    .sort((a, b) => a.created_ts - b.created_ts);
  return conversation;
};

// Immutably replace a conversation in the map.
const withConversation = (
  byId: Record<string, Conversation>,
  conversation: Conversation
): Record<string, Conversation> => ({
  ...byId,
  [conversation.id]: conversation,
});

export const useConversationStore = create<ConversationState>((set, get) => {
  const deleteConversation = async (id: string): Promise<void> => {
    const conversation = get().conversationsById[id];
    if (!conversation) return;
    if (conversation.messageList.length > 0) {
      await messages.bulkDocs(
        conversation.messageList.map((message) => ({
          ...convertMessageToEntity(message),
          row_status: "ARCHIVED" as const,
        }))
      );
    }
    await conversations.put(
      { ...convertConversationToEntity(conversation), row_status: "ARCHIVED" },
      { force: true }
    );
    set((state) => {
      const next = { ...state.conversationsById };
      delete next[id];
      return { conversationsById: next };
    });
  };

  const updateMessage = async (message: Message): Promise<Message> => {
    await messages.put(convertMessageToEntity(message), { force: true });
    const conversation = get().conversationsById[message.conversation.id];
    if (conversation) {
      const nextConversation: Conversation = {
        ...conversation,
        messageList: conversation.messageList.map((m) =>
          m.id === message.id ? message : m
        ),
      };
      set((state) => ({
        conversationsById: withConversation(
          state.conversationsById,
          nextConversation
        ),
      }));
    }
    return message;
  };

  const fixAbnormalMessages = async (messageList: Message[]) => {
    const requests = messageList
      .filter((message) => message.status === "LOADING")
      .map((message) =>
        updateMessage({
          ...message,
          status: "FAILED",
          error: "Request timeout",
        })
      );
    await Promise.all(requests);
  };

  const fetchConversationListByConnection = async (
    conn: SQLEditorConnection
  ): Promise<Conversation[]> => {
    const conversationEntityList = (
      await conversations.find({
        selector: {
          row_status: { $eq: "NORMAL" },
          instance: { $eq: conn.instance },
          database: { $eq: conn.database },
        },
      })
    ).docs;
    const flattenMessageList = (
      await messages.find({
        selector: {
          row_status: { $eq: "NORMAL" },
          conversation_id: { $in: conversationEntityList.map((c) => c._id) },
        },
      })
    ).docs;

    const groupByConversationId = groupBy(
      flattenMessageList,
      (m) => m.conversation_id
    );
    conversationEntityList.sort((a, b) => a.created_ts - b.created_ts);
    const rawConversationList = conversationEntityList.map<Conversation>((c) =>
      buildConversation(c, groupByConversationId[c._id] ?? [])
    );

    await fixAbnormalMessages(
      rawConversationList.flatMap((c) => c.messageList)
    );

    const emptyConversationList = rawConversationList.filter(
      (c) => c.messageList.length === 0
    );

    set((state) => {
      const next = { ...state.conversationsById };
      for (const c of rawConversationList) next[c.id] = c;
      return {
        conversationsById: next,
        readyByConnection: {
          ...state.readyByConnection,
          [connectionKey(conn)]: true,
        },
      };
    });

    await Promise.all(
      emptyConversationList.map((c) => deleteConversation(c.id))
    );

    return rawConversationList.filter((c) => c.messageList.length > 0);
  };

  const createConversation = async (
    conversationCreate: EntityCreate<ConversationEntity>
  ): Promise<Conversation> => {
    const c: ConversationEntity = {
      _id: uuidv1(),
      created_ts: Date.now(),
      row_status: "NORMAL",
      ...conversationCreate,
    };
    const response = await conversations.put(c);
    c._rev = response.rev;
    const conversation = buildConversation(c, []);
    set((state) => ({
      conversationsById: withConversation(
        state.conversationsById,
        conversation
      ),
    }));
    return conversation;
  };

  const updateConversation = async (
    conversation: Conversation
  ): Promise<Conversation> => {
    await conversations.put(convertConversationToEntity(conversation), {
      force: true,
    });
    const existing = get().conversationsById[conversation.id];
    const next: Conversation = {
      ...conversation,
      messageList: existing?.messageList ?? conversation.messageList,
    };
    set((state) => ({
      conversationsById: withConversation(state.conversationsById, next),
    }));
    return next;
  };

  const createMessage = async (
    messageCreate: EntityCreate<MessageEntity>
  ): Promise<Message> => {
    const m: MessageEntity = {
      _id: uuidv1(),
      created_ts: Date.now(),
      row_status: "NORMAL",
      ...messageCreate,
    };
    const response = await messages.put(m);
    m._rev = response.rev;
    const conversation = get().conversationsById[m.conversation_id];
    const message: Message = {
      ...omit(m, "_id", "_rev", "row_status", "conversation_id"),
      id: m._id,
      conversation,
    } as Message;
    if (conversation) {
      const nextConversation: Conversation = {
        ...conversation,
        messageList: [...conversation.messageList, message],
      };
      message.conversation = nextConversation;
      set((state) => ({
        conversationsById: withConversation(
          state.conversationsById,
          nextConversation
        ),
      }));
    }
    return message;
  };

  const reset = async () => {
    try {
      await Promise.all(ready);
      await Promise.all([conversations.destroy(), messages.destroy()]);
    } catch {
      // nothing to do
    }
    set({ conversationsById: {}, readyByConnection: {} });
  };

  return {
    conversationsById: {},
    readyByConnection: {},
    fetchConversationListByConnection,
    createConversation,
    updateConversation,
    deleteConversation,
    createMessage,
    updateMessage,
    reset,
  };
});

// Conversations for a given connection, sorted by creation time (mirrors the
// Vue store's filtered `conversationList`).
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
