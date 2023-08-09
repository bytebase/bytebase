import { groupBy, omit } from "lodash-es";
import { defineStore } from "pinia";
import PouchDB from "pouchdb";
import PouchDBFind from "pouchdb-find";
import { v1 as uuidv1 } from "uuid";
import { computed, reactive, ref, unref, watchEffect } from "vue";
import { Connection, MaybeRef } from "@/types";
import { Conversation, Message } from "../types";

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

const convertConversationToEntity = (conversation: Conversation) => {
  const c: ConversationEntity = {
    ...omit(conversation, "id", "messageList"),
    _id: conversation.id,
    row_status: "NORMAL",
  };
  return c;
};

const convertMessageToEntity = (message: Message) => {
  const m: MessageEntity = {
    ...omit(message, "id", "conversation"),
    _id: message.id,
    row_status: "NORMAL",
    conversation_id: message.conversation.id,
  };
  return m;
};

const useLocalCache = () => {
  const conversationById = reactive(new Map<string, Conversation>());
  const messageById = reactive(new Map<string, Message>());

  const convertConversation = (
    c: ConversationEntity,
    messageList: Message[] = []
  ): Conversation => {
    const existed = conversationById.get(c._id);
    if (existed) {
      Object.assign(existed, {
        ...c,
        messageList,
      });
      return existed;
    }
    const conversation = reactive({
      ...c,
      id: c._id,
      messageList,
    });
    conversationById.set(conversation.id, conversation);
    return conversation;
  };

  const convertMessage = (
    m: MessageEntity,
    conversation: Conversation
  ): Message => {
    const existed = messageById.get(m._id);
    if (existed) {
      Object.assign(existed, {
        ...m,
        conversation,
      });
    }
    const message = reactive({
      ...m,
      id: m._id,
      conversation,
    });
    messageById.set(message.id, message);
    return message;
  };

  const getConversationById = (id: string) => {
    return conversationById.get(id)!;
  };
  const getMessageById = (id: string) => {
    return messageById.get(id)!;
  };

  return {
    conversationById,
    convertConversation,
    convertMessage,
    getConversationById,
    getMessageById,
  };
};

const FK_MESSAGE_CONVERSATION_ID = "fk_message_conversation_id";

export const useConversationStore = defineStore("ai-conversation", () => {
  const conversations = new PouchDB<ConversationEntity>(
    "bb.plugin.ai.conversations"
  );
  const messages = new PouchDB<MessageEntity>("bb.plugin.ai.messages");
  const ready: Promise<any>[] = [];
  ready.push(
    messages.createIndex({
      index: { name: FK_MESSAGE_CONVERSATION_ID, fields: ["conversation_id"] },
    })
  );

  const {
    conversationById,
    convertConversation,
    convertMessage,
    getConversationById,
  } = useLocalCache();

  const conversationList = computed(() => {
    return [...conversationById.values()];
  });

  const fetchConversationListByConnection = async (conn: Connection) => {
    const conversationEntityList = (
      await conversations.find({
        selector: {
          row_status: { $eq: "NORMAL" },
          instanceId: { $eq: conn.instanceId },
          databaseId: { $eq: conn.databaseId },
        },
      })
    ).docs;
    const flattenMessageMessageList = (
      await messages.find({
        selector: {
          row_status: { $eq: "NORMAL" },
          conversation_id: {
            $in: conversationEntityList.map((c) => c._id),
          },
        },
      })
    ).docs;

    const groupByConversationId = groupBy(
      flattenMessageMessageList,
      (m) => m.conversation_id
    );
    conversationEntityList.sort((a, b) => a.created_ts - b.created_ts);
    const conversationList = conversationEntityList.map<Conversation>((c) => {
      const conversation = convertConversation(c);
      const messageEntityList = groupByConversationId[c._id] ?? [];
      conversation.messageList = messageEntityList.map((m) =>
        convertMessage(m, conversation)
      );
      conversation.messageList.sort((a, b) => a.created_ts - b.created_ts);
      return conversation;
    });
    await fixAbnormalMessages(conversationList.flatMap((c) => c.messageList));
    return conversationList;
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
    return convertConversation(c);
  };

  const updateConversation = async (conversation: Conversation) => {
    const c = convertConversationToEntity(conversation);
    await conversations.put(c, { force: true });
    return convertConversation(c, conversation.messageList);
  };

  const deleteConversation = async (id: string) => {
    const conversation = getConversationById(id);
    if (conversation.messageList.length > 0) {
      await messages.bulkDocs(
        conversation.messageList.map((message) => ({
          ...convertMessageToEntity(message),
          row_status: "ARCHIVED",
        }))
      );
    }
    await conversations.put(
      {
        ...convertConversationToEntity(conversation),
        row_status: "ARCHIVED",
      },
      {
        force: true,
      }
    );
    conversationById.delete(conversation.id);
  };

  const createMessage = async (messageCreate: EntityCreate<MessageEntity>) => {
    const m: MessageEntity = {
      _id: uuidv1(),
      created_ts: Date.now(),
      row_status: "NORMAL",
      ...messageCreate,
    };
    const response = await messages.put(m);
    m._rev = response.rev;
    const conversation = getConversationById(m.conversation_id);
    const message = convertMessage(m, conversation);
    conversation.messageList.push(message);
    return message;
  };

  const updateMessage = async (message: Message) => {
    const m = convertMessageToEntity(message);
    await messages.put(m, {
      force: true,
    });
    return convertMessage(m, message.conversation);
  };

  const fixAbnormalMessages = async (messageList: Message[]) => {
    const requests = messageList
      .filter((message) => message.status === "LOADING")
      .map((message) => {
        message.status = "FAILED";
        message.error = "Request timeout";
        return updateMessage(message);
      });
    await Promise.all(requests);
  };

  const reset = async () => {
    try {
      await Promise.all(ready);
      await Promise.all([conversations.destroy(), messages.destroy()]);
    } catch (ex) {
      // nothing todo
    }
  };

  return {
    conversationById,
    conversationList,
    fetchConversationListByConnection,
    createConversation,
    updateConversation,
    deleteConversation,
    createMessage,
    updateMessage,
    reset,
  };
});

export const useConversationListByConnection = (conn: MaybeRef<Connection>) => {
  const store = useConversationStore();
  const ready = ref(false);
  watchEffect(async () => {
    ready.value = false;
    await store.fetchConversationListByConnection(unref(conn));
    ready.value = true;
  });
  const list = computed(() => {
    const { instanceId, databaseId } = unref(conn);
    return store.conversationList.filter(
      (c) => c.instanceId === instanceId && c.databaseId === databaseId
    );
  });
  return { list, ready };
};
