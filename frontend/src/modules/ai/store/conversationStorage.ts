import type {
  Conversation,
  Message,
  MessageAuthor,
  MessageStatus,
} from "../types";

const STORAGE_VERSION = 1;
const MAX_CONVERSATIONS = 20;
export const MAX_CONVERSATION_HISTORY_BYTES = 1024 * 1024;

type StoredMessage = {
  id: string;
  created_ts: number;
  author: MessageAuthor;
  content: string;
  status: MessageStatus;
  error: string;
};

type StoredConversation = Omit<Conversation, "messageList"> & {
  messageList: StoredMessage[];
};

type StoredConversationHistory = {
  version: typeof STORAGE_VERSION;
  conversations: StoredConversation[];
};

const serializedSize = (value: string) =>
  new TextEncoder().encode(value).byteLength;

const serialize = (conversations: StoredConversation[]) =>
  JSON.stringify({
    version: STORAGE_VERSION,
    conversations,
  } satisfies StoredConversationHistory);

const removeOldestMessages = (conversation: StoredConversation) => {
  conversation.messageList.shift();
  while (
    conversation.messageList.length > 0 &&
    conversation.messageList[0]?.author !== "USER"
  ) {
    conversation.messageList.shift();
  }
};

const fitToLimit = (conversations: StoredConversation[]) => {
  const bounded = conversations
    .sort((a, b) => a.created_ts - b.created_ts)
    .slice(-MAX_CONVERSATIONS);
  let serialized = serialize(bounded);
  while (serializedSize(serialized) > MAX_CONVERSATION_HISTORY_BYTES) {
    if (bounded.length > 1) {
      bounded.shift();
    } else if (bounded[0]?.messageList.length) {
      removeOldestMessages(bounded[0]);
    } else if (bounded.length === 1) {
      bounded.shift();
    } else {
      break;
    }
    serialized = serialize(bounded);
  }
  return { conversations: bounded, serialized };
};

const forRetry = (conversations: StoredConversation[]) => {
  if (conversations.length > 1) {
    return conversations.slice(Math.floor(conversations.length / 2));
  }
  const [conversation] = conversations;
  if (!conversation || conversation.messageList.length === 0) {
    return conversations;
  }
  const retryConversation = {
    ...conversation,
    messageList: conversation.messageList.slice(
      Math.ceil(conversation.messageList.length / 2)
    ),
  };
  while (
    retryConversation.messageList.length > 0 &&
    retryConversation.messageList[0]?.author !== "USER"
  ) {
    retryConversation.messageList.shift();
  }
  return [retryConversation];
};

const toStoredConversation = (
  conversation: Conversation
): StoredConversation => ({
  id: conversation.id,
  created_ts: conversation.created_ts,
  name: conversation.name,
  instance: conversation.instance,
  database: conversation.database,
  messageList: conversation.messageList.map(
    ({ id, created_ts, author, content, status, error }) => ({
      id,
      created_ts,
      author,
      content,
      status,
      error,
    })
  ),
});

const isStoredMessage = (value: unknown): value is StoredMessage => {
  if (!value || typeof value !== "object") return false;
  const message = value as Partial<StoredMessage>;
  return (
    typeof message.id === "string" &&
    typeof message.created_ts === "number" &&
    (message.author === "USER" || message.author === "AI") &&
    typeof message.content === "string" &&
    ["LOADING", "DONE", "FAILED"].includes(message.status ?? "") &&
    typeof message.error === "string"
  );
};

const isStoredConversation = (value: unknown): value is StoredConversation => {
  if (!value || typeof value !== "object") return false;
  const conversation = value as Partial<StoredConversation>;
  return (
    typeof conversation.id === "string" &&
    typeof conversation.created_ts === "number" &&
    typeof conversation.name === "string" &&
    typeof conversation.instance === "string" &&
    typeof conversation.database === "string" &&
    Array.isArray(conversation.messageList) &&
    conversation.messageList.every(isStoredMessage)
  );
};

export const loadConversationHistory = (key: string): Conversation[] => {
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as Partial<StoredConversationHistory>;
    if (
      parsed.version !== STORAGE_VERSION ||
      !Array.isArray(parsed.conversations) ||
      !parsed.conversations.every(isStoredConversation)
    ) {
      localStorage.removeItem(key);
      return [];
    }
    return parsed.conversations.map((stored) => {
      const conversation: Conversation = {
        id: stored.id,
        created_ts: stored.created_ts,
        name: stored.name,
        instance: stored.instance,
        database: stored.database,
        messageList: [],
      };
      conversation.messageList = stored.messageList.map<Message>((message) => ({
        ...message,
        status: message.status === "LOADING" ? "FAILED" : message.status,
        error: message.status === "LOADING" ? "Request timeout" : message.error,
        conversation,
      }));
      return conversation;
    });
  } catch {
    return [];
  }
};

export const saveConversationHistory = (
  key: string,
  conversations: Conversation[]
): boolean => {
  const stored = conversations
    .filter((conversation) => conversation.messageList.length > 0)
    .map(toStoredConversation);
  const bounded = fitToLimit(stored);
  try {
    localStorage.setItem(key, bounded.serialized);
    return true;
  } catch {
    try {
      const retry = fitToLimit(forRetry(bounded.conversations));
      localStorage.setItem(key, retry.serialized);
      return true;
    } catch {
      return false;
    }
  }
};
