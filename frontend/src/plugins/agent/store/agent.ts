import { defineStore } from "pinia";
import { v4 as uuidv4 } from "uuid";
import { computed, ref, watch } from "vue";
import type {
  AgentAskUserResponse,
  AgentChat,
  AgentChatSnapshot,
  AgentChatStatus,
  AgentMessage,
  AgentPendingAsk,
  Message,
  ToolCall,
} from "../logic/types";

export const AGENT_STATE_KEY = "bb-agent-state-v2";
export const AGENT_WINDOW_KEY = "bb-agent-window";

interface PersistedAgentState {
  currentChatId: string | null;
  chats: AgentChat[];
  messagesByChatId: Record<string, AgentMessage[]>;
  pendingAskByChatId: Record<string, AgentPendingAsk>;
}

interface CreateChatOptions {
  page?: AgentChatSnapshot;
  select?: boolean;
  title?: string;
  archived?: boolean;
}

interface AddMessageOptions extends Message {
  chatId?: string;
  metadata?: AgentMessage["metadata"];
}

const DEFAULT_CHAT_STATUS: AgentChatStatus = "idle";

const createChatRecord = (options: CreateChatOptions = {}): AgentChat => {
  const now = Date.now();
  return {
    id: uuidv4(),
    title: options.title ?? "",
    createdTs: now,
    updatedTs: now,
    status: DEFAULT_CHAT_STATUS,
    totalTokensUsed: 0,
    page: options.page,
    archived: options.archived ?? false,
    lastError: null,
    interrupted: false,
    runId: null,
  };
};

const getChatTitleFromMessage = (content?: string) => {
  const title = (content ?? "").trim().replace(/\s+/g, " ");
  if (!title) {
    return "";
  }
  return title.length <= 48 ? title : `${title.slice(0, 45)}...`;
};

const isRecord = (value: unknown): value is Record<string, unknown> => {
  return typeof value === "object" && value !== null;
};

const normalizeMessage = (
  raw: unknown,
  chatId: string,
  fallbackTs: number
): AgentMessage => {
  const message = isRecord(raw) ? raw : {};
  return {
    id: typeof message.id === "string" ? message.id : uuidv4(),
    chatId:
      typeof message.chatId === "string"
        ? message.chatId
        : typeof message.threadId === "string"
          ? message.threadId
          : chatId,
    createdTs:
      typeof message.createdTs === "number" ? message.createdTs : fallbackTs,
    role:
      message.role === "assistant" ||
      message.role === "system" ||
      message.role === "tool"
        ? message.role
        : "user",
    content: typeof message.content === "string" ? message.content : undefined,
    toolCalls: Array.isArray(message.toolCalls)
      ? (message.toolCalls as ToolCall[])
      : undefined,
    toolCallId:
      typeof message.toolCallId === "string" ? message.toolCallId : undefined,
    metadata: isRecord(message.metadata)
      ? { ...(message.metadata as AgentMessage["metadata"]) }
      : undefined,
  };
};

const normalizePendingAsk = (raw: unknown): AgentPendingAsk | null => {
  const pendingAsk = isRecord(raw) ? raw : {};
  if (
    typeof pendingAsk.toolCallId !== "string" ||
    typeof pendingAsk.prompt !== "string"
  ) {
    return null;
  }

  const options = Array.isArray(pendingAsk.options)
    ? pendingAsk.options
        .map((option) => {
          const candidate = isRecord(option) ? option : {};
          const label =
            typeof candidate.label === "string"
              ? candidate.label.trim()
              : typeof candidate.value === "string"
                ? candidate.value.trim()
                : "";
          const value =
            typeof candidate.value === "string" ? candidate.value.trim() : "";
          if (!label || !value) {
            return null;
          }
          return {
            label,
            value,
            description:
              typeof candidate.description === "string" &&
              candidate.description.trim()
                ? candidate.description.trim()
                : undefined,
          };
        })
        .filter(
          (option): option is NonNullable<typeof option> => option !== null
        )
    : [];

  const kind =
    pendingAsk.kind === "confirm"
      ? "confirm"
      : pendingAsk.kind === "choose" && options.length > 0
        ? "choose"
        : "input";

  return {
    toolCallId: pendingAsk.toolCallId,
    prompt: pendingAsk.prompt,
    kind,
    defaultValue:
      typeof pendingAsk.defaultValue === "string"
        ? pendingAsk.defaultValue
        : undefined,
    confirmLabel:
      typeof pendingAsk.confirmLabel === "string"
        ? pendingAsk.confirmLabel
        : undefined,
    cancelLabel:
      typeof pendingAsk.cancelLabel === "string"
        ? pendingAsk.cancelLabel
        : undefined,
    options: kind === "choose" ? options : undefined,
  };
};

const normalizeChat = (raw: unknown): AgentChat => {
  const chat = isRecord(raw) ? raw : {};
  const now = Date.now();
  const status =
    chat.status === "running" ||
    chat.status === "awaiting_user" ||
    chat.status === "error"
      ? chat.status
      : DEFAULT_CHAT_STATUS;
  return {
    id: typeof chat.id === "string" ? chat.id : uuidv4(),
    title: typeof chat.title === "string" ? chat.title : "",
    createdTs: typeof chat.createdTs === "number" ? chat.createdTs : now,
    updatedTs: typeof chat.updatedTs === "number" ? chat.updatedTs : now,
    status,
    totalTokensUsed:
      typeof chat.totalTokensUsed === "number" &&
      Number.isFinite(chat.totalTokensUsed) &&
      chat.totalTokensUsed >= 0
        ? chat.totalTokensUsed
        : 0,
    page:
      isRecord(chat.page) &&
      typeof chat.page.path === "string" &&
      typeof chat.page.title === "string"
        ? {
            path: chat.page.path,
            title: chat.page.title,
          }
        : undefined,
    archived: Boolean(chat.archived),
    lastError:
      typeof chat.lastError === "string"
        ? chat.lastError
        : chat.lastError === null
          ? null
          : null,
    interrupted: Boolean(chat.interrupted),
    runId: typeof chat.runId === "string" ? chat.runId : null,
  };
};

const sortMessages = (messages: AgentMessage[]) => {
  messages.sort((a, b) => a.createdTs - b.createdTs);
  return messages;
};

const createEmptyPersistedState = (): PersistedAgentState => ({
  currentChatId: null,
  chats: [],
  messagesByChatId: {},
  pendingAskByChatId: {},
});

const normalizePersistedState = (raw: unknown): PersistedAgentState => {
  if (!isRecord(raw)) {
    return createEmptyPersistedState();
  }

  const rawChats = Array.isArray(raw.chats)
    ? raw.chats
    : Array.isArray(raw.threads)
      ? raw.threads
      : [];
  const rawMessagesByChatId = isRecord(raw.messagesByChatId)
    ? raw.messagesByChatId
    : isRecord(raw.messagesByThreadId)
      ? raw.messagesByThreadId
      : {};
  const rawPendingAskByChatId = isRecord(raw.pendingAskByChatId)
    ? raw.pendingAskByChatId
    : isRecord(raw.pendingAskByThreadId)
      ? raw.pendingAskByThreadId
      : {};

  const chats = rawChats.map((chat) => normalizeChat(chat));
  const messagesByChatId: Record<string, AgentMessage[]> = {};
  const pendingAskByChatId: Record<string, AgentPendingAsk> = {};

  for (const chat of chats) {
    const rawMessages: unknown[] = Array.isArray(rawMessagesByChatId[chat.id])
      ? (rawMessagesByChatId[chat.id] as unknown[])
      : [];
    const messages = sortMessages(
      rawMessages.map((message, index) =>
        normalizeMessage(message, chat.id, chat.createdTs + index)
      )
    );

    const firstUserMessage = messages.find(
      (message) => message.role === "user"
    );
    const lastMessage = messages.at(-1);

    if (!chat.title) {
      chat.title = getChatTitleFromMessage(firstUserMessage?.content);
    }
    if (lastMessage) {
      chat.updatedTs = Math.max(chat.updatedTs, lastMessage.createdTs);
    }
    if (chat.status === "running") {
      chat.status = DEFAULT_CHAT_STATUS;
      chat.interrupted = true;
      chat.lastError = null;
    }

    const pendingAsk = normalizePendingAsk(rawPendingAskByChatId[chat.id]);
    if (chat.status === "awaiting_user") {
      if (pendingAsk) {
        pendingAskByChatId[chat.id] = pendingAsk;
      } else {
        chat.status = DEFAULT_CHAT_STATUS;
      }
    }

    messagesByChatId[chat.id] = messages;
  }

  const savedCurrentChatId =
    typeof raw.currentChatId === "string"
      ? raw.currentChatId
      : typeof raw.currentThreadId === "string"
        ? raw.currentThreadId
        : null;
  const currentChatId =
    savedCurrentChatId && chats.some((chat) => chat.id === savedCurrentChatId)
      ? savedCurrentChatId
      : (chats[0]?.id ?? null);

  return {
    currentChatId,
    chats,
    messagesByChatId,
    pendingAskByChatId,
  };
};

export const useAgentStore = defineStore("agent", () => {
  const visible = ref(false);
  const position = ref({
    x: window.innerWidth - 420,
    y: window.innerHeight - 520,
  });
  const size = ref({ width: 400, height: 500 });
  const sidebarWidth = ref(256);
  const minimized = ref(false);

  const chats = ref<AgentChat[]>([]);
  const messagesByChatId = ref<Record<string, AgentMessage[]>>({});
  const pendingAskByChatId = ref<Record<string, AgentPendingAsk>>({});
  const currentChatId = ref<string | null>(null);
  const abortControllersByChatId = ref<Record<string, AbortController>>({});

  const orderedChats = computed(() =>
    [...chats.value].sort((a, b) => {
      if (b.updatedTs !== a.updatedTs) {
        return b.updatedTs - a.updatedTs;
      }
      return b.createdTs - a.createdTs;
    })
  );
  const currentChat = computed(
    () => chats.value.find((chat) => chat.id === currentChatId.value) ?? null
  );
  const messages = computed(
    () =>
      (currentChatId.value
        ? messagesByChatId.value[currentChatId.value]
        : undefined) ?? []
  );
  const currentPendingAsk = computed(
    () =>
      (currentChatId.value
        ? pendingAskByChatId.value[currentChatId.value]
        : undefined) ?? null
  );
  const loading = computed(() => currentChat.value?.status === "running");
  const error = computed(() => currentChat.value?.lastError ?? null);
  const runningChatIds = computed(() =>
    chats.value
      .filter((chat) => chat.status === "running")
      .map((chat) => chat.id)
  );
  const hasRunningChat = computed(() => runningChatIds.value.length > 0);

  const getChat = (chatId?: string | null) => {
    if (!chatId) {
      return null;
    }
    return chats.value.find((chat) => chat.id === chatId) ?? null;
  };

  const getMessages = (chatId?: string | null) => {
    if (!chatId) {
      return [];
    }
    return messagesByChatId.value[chatId] ?? [];
  };

  const getPendingAsk = (chatId = currentChatId.value) => {
    if (!chatId) {
      return null;
    }
    return pendingAskByChatId.value[chatId] ?? null;
  };

  const getAbortController = (chatId?: string | null) => {
    if (!chatId) {
      return null;
    }
    return abortControllersByChatId.value[chatId] ?? null;
  };

  const isChatRunning = (chatId?: string | null) => {
    return getChat(chatId)?.status === "running";
  };

  const touchChat = (chatId: string) => {
    const chat = getChat(chatId);
    if (!chat) {
      return null;
    }
    chat.updatedTs = Date.now();
    return chat;
  };

  const setAbortController = (
    chatId: string,
    controller: AbortController | null
  ) => {
    if (controller) {
      abortControllersByChatId.value[chatId] = controller;
      return controller;
    }
    delete abortControllersByChatId.value[chatId];
    return null;
  };

  const clearPendingAsk = (chatId = currentChatId.value) => {
    if (!chatId || !pendingAskByChatId.value[chatId]) {
      return;
    }
    delete pendingAskByChatId.value[chatId];
    touchChat(chatId);
  };

  const setPendingAsk = (chatId: string, pendingAsk: AgentPendingAsk) => {
    pendingAskByChatId.value[chatId] = pendingAsk;
    touchChat(chatId);
    return pendingAsk;
  };

  const setChatStatus = (
    chatId: string,
    status: AgentChatStatus,
    options: {
      interrupted?: boolean;
      lastError?: string | null;
      page?: AgentChatSnapshot;
    } = {}
  ) => {
    const chat = getChat(chatId);
    if (!chat) {
      return null;
    }
    chat.status = status;
    chat.interrupted = options.interrupted ?? false;
    chat.lastError = options.lastError ?? null;
    if (options.page) {
      chat.page = options.page;
    }
    if (status !== "awaiting_user") {
      delete pendingAskByChatId.value[chatId];
    }
    touchChat(chatId);
    return chat;
  };

  const ensureCurrentChat = (page?: AgentChatSnapshot) => {
    const existing = getChat(currentChatId.value);
    if (existing) {
      if (page && !existing.page) {
        existing.page = page;
        touchChat(existing.id);
      }
      return existing;
    }
    return createChat({ page });
  };

  const createChat = (options: CreateChatOptions = {}) => {
    const chat = createChatRecord(options);
    chats.value.push(chat);
    messagesByChatId.value[chat.id] = [];
    const shouldSelect =
      (options.select ?? true) &&
      (!hasRunningChat.value || currentChatId.value === null);
    if (shouldSelect) {
      currentChatId.value = chat.id;
    }
    return chat;
  };

  const selectNextAvailableChat = (preferredChatId?: string | null) => {
    const preferredChat = getChat(preferredChatId);
    if (preferredChat) {
      currentChatId.value = preferredChat.id;
      return preferredChat;
    }
    const fallbackChat = orderedChats.value[0] ?? createChat();
    currentChatId.value = fallbackChat.id;
    return fallbackChat;
  };

  const canSelectChat = (chatId?: string | null) => {
    if (!chatId || !getChat(chatId)) {
      return false;
    }
    return !hasRunningChat.value || currentChatId.value === chatId;
  };

  const setCurrentChat = (chatId: string) => {
    if (canSelectChat(chatId)) {
      currentChatId.value = chatId;
    }
  };

  const updateChatPage = (chatId: string, page: AgentChatSnapshot) => {
    const chat = getChat(chatId);
    if (!chat) {
      return null;
    }
    chat.page = page;
    touchChat(chatId);
    return chat;
  };

  const renameChat = (chatId: string, title: string) => {
    const chat = getChat(chatId);
    if (!chat) {
      return null;
    }
    chat.title = title.trim();
    touchChat(chatId);
    return chat;
  };

  const archiveChat = (chatId: string) => {
    const chat = getChat(chatId);
    if (!chat) {
      return null;
    }
    chat.archived = true;
    touchChat(chatId);
    return chat;
  };

  const unarchiveChat = (chatId: string) => {
    const chat = getChat(chatId);
    if (!chat) {
      return null;
    }
    chat.archived = false;
    touchChat(chatId);
    return chat;
  };

  const addMessage = (message: AddMessageOptions) => {
    const chat = getChat(message.chatId) ?? ensureCurrentChat();
    const createdTs = Date.now();
    const agentMessage: AgentMessage = {
      ...message,
      id: uuidv4(),
      chatId: chat.id,
      createdTs,
    };
    const chatMessages = getMessages(chat.id);
    if (chatMessages.length === 0) {
      messagesByChatId.value[chat.id] = chatMessages;
    }
    chatMessages.push(agentMessage);
    if (message.role === "user" && !chat.title) {
      chat.title = getChatTitleFromMessage(message.content);
    }
    chat.lastError = null;
    chat.interrupted = false;
    chat.updatedTs = createdTs;
    return agentMessage;
  };

  const removeMessagesByRunId = (chatId: string, runId?: string | null) => {
    if (!runId) {
      return [];
    }
    const chat = getChat(chatId);
    if (!chat) {
      return [];
    }
    const existingMessages = getMessages(chatId);
    const removedMessages = existingMessages.filter(
      (message) => message.metadata?.runId === runId
    );
    if (removedMessages.length === 0) {
      return [];
    }
    messagesByChatId.value[chatId] = existingMessages.filter(
      (message) => message.metadata?.runId !== runId
    );
    touchChat(chatId);
    return removedMessages;
  };

  const appendToolCall = (
    chatId: string,
    messageId: string,
    toolCall: ToolCall
  ) => {
    const message = getMessages(chatId).find(
      (candidate) => candidate.id === messageId
    );
    if (!message) {
      return null;
    }
    message.toolCalls = [...(message.toolCalls ?? []), toolCall];
    touchChat(chatId);
    return message;
  };

  const incrementChatTotalTokens = (
    chatId: string,
    totalTokensUsed: number
  ) => {
    const chat = getChat(chatId);
    if (!chat || totalTokensUsed <= 0) {
      return null;
    }
    chat.totalTokensUsed += totalTokensUsed;
    touchChat(chatId);
    return chat;
  };

  const awaitUser = (chatId: string, pendingAsk: AgentPendingAsk) => {
    setChatStatus(chatId, "awaiting_user");
    return setPendingAsk(chatId, pendingAsk);
  };

  const answerPendingAsk = (
    chatId: string,
    response: AgentAskUserResponse,
    metadata?: AgentMessage["metadata"]
  ) => {
    const pendingAsk = getPendingAsk(chatId);
    if (!pendingAsk) {
      return null;
    }
    const toolMessage = addMessage({
      chatId,
      role: "tool",
      toolCallId: pendingAsk.toolCallId,
      content: JSON.stringify(response),
      metadata,
    });
    clearPendingAsk(chatId);
    return toolMessage;
  };

  const cancel = (chatId = currentChatId.value) => {
    if (!chatId) {
      return;
    }
    getAbortController(chatId)?.abort();
    setAbortController(chatId, null);
    if (isChatRunning(chatId)) {
      interruptChatRun(chatId);
    }
  };

  const clearError = (chatId = currentChatId.value) => {
    const chat = getChat(chatId);
    if (!chat) {
      return;
    }
    chat.lastError = null;
    chat.interrupted = false;
    chat.runId = null;
    if (chat.status === "error") {
      chat.status = DEFAULT_CHAT_STATUS;
    }
    touchChat(chat.id);
  };

  const interruptChatRun = (chatId: string, page?: AgentChatSnapshot) => {
    setChatStatus(chatId, DEFAULT_CHAT_STATUS, {
      interrupted: true,
      page,
    });
  };

  const startChatRun = (
    chatId: string,
    page?: AgentChatSnapshot,
    options: {
      runId?: string;
    } = {}
  ) => {
    setChatStatus(chatId, "running", {
      page,
    });
    const chat = getChat(chatId);
    if (chat) {
      chat.runId = options.runId ?? chat.runId ?? null;
    }
  };

  const finishChatRun = (
    chatId: string,
    options: {
      status?: Extract<AgentChatStatus, "idle" | "error">;
      lastError?: string | null;
    } = {}
  ) => {
    setChatStatus(chatId, options.status ?? DEFAULT_CHAT_STATUS, {
      lastError: options.lastError ?? null,
      interrupted: false,
    });
    const chat = getChat(chatId);
    if (chat) {
      chat.runId = null;
    }
  };

  const deleteChat = (chatId: string) => {
    if (!getChat(chatId)) {
      return false;
    }
    cancel(chatId);
    chats.value = chats.value.filter((chat) => chat.id !== chatId);
    delete messagesByChatId.value[chatId];
    delete pendingAskByChatId.value[chatId];
    delete abortControllersByChatId.value[chatId];
    if (currentChatId.value === chatId) {
      selectNextAvailableChat();
    }
    return true;
  };

  const saveWindowState = () => {
    localStorage.setItem(
      AGENT_WINDOW_KEY,
      JSON.stringify({
        position: position.value,
        size: size.value,
        sidebarWidth: sidebarWidth.value,
      })
    );
  };

  const saveState = () => {
    const persistedState: PersistedAgentState = {
      currentChatId: currentChatId.value,
      chats: chats.value,
      messagesByChatId: messagesByChatId.value,
      pendingAskByChatId: pendingAskByChatId.value,
    };
    localStorage.setItem(AGENT_STATE_KEY, JSON.stringify(persistedState));
  };

  const loadState = () => {
    const saved = localStorage.getItem(AGENT_STATE_KEY);
    if (saved) {
      try {
        const state = normalizePersistedState(JSON.parse(saved));
        chats.value = state.chats;
        messagesByChatId.value = state.messagesByChatId;
        pendingAskByChatId.value = state.pendingAskByChatId;
        currentChatId.value = state.currentChatId;
      } catch {
        localStorage.removeItem(AGENT_STATE_KEY);
      }
    }

    if (!currentChatId.value || !getChat(currentChatId.value)) {
      const chat = createChat();
      currentChatId.value = chat.id;
    }
  };

  const loadWindowState = () => {
    const saved = localStorage.getItem(AGENT_WINDOW_KEY);
    if (!saved) {
      return;
    }
    try {
      const state = JSON.parse(saved) as {
        position?: { x?: number; y?: number };
        size?: { width?: number; height?: number };
        sidebarWidth?: number;
      };
      if (
        typeof state.position?.x === "number" &&
        typeof state.position?.y === "number"
      ) {
        position.value = {
          x: state.position.x,
          y: state.position.y,
        };
      }
      if (
        typeof state.size?.width === "number" &&
        typeof state.size?.height === "number"
      ) {
        size.value = {
          width: state.size.width,
          height: state.size.height,
        };
      }
      if (typeof state.sidebarWidth === "number") {
        sidebarWidth.value = state.sidebarWidth;
      }
    } catch {
      localStorage.removeItem(AGENT_WINDOW_KEY);
    }
  };

  watch(
    [chats, messagesByChatId, pendingAskByChatId, currentChatId],
    saveState,
    {
      deep: true,
    }
  );

  loadState();

  return {
    visible,
    position,
    size,
    sidebarWidth,
    minimized,
    chats,
    orderedChats,
    currentChatId,
    currentChat,
    messages,
    currentPendingAsk,
    loading,
    error,
    runningChatIds,
    hasRunningChat,
    abortControllersByChatId,
    toggle() {
      visible.value = !visible.value;
      if (visible.value) {
        minimized.value = false;
      }
    },
    minimize() {
      minimized.value = true;
    },
    restore() {
      minimized.value = false;
    },
    getChat,
    getMessages,
    getPendingAsk,
    getAbortController,
    isChatRunning,
    canSelectChat,
    setAbortController,
    createChat,
    ensureCurrentChat,
    setCurrentChat,
    updateChatPage,
    renameChat,
    archiveChat,
    unarchiveChat,
    deleteChat,
    setPendingAsk,
    clearPendingAsk,
    setChatStatus,
    startChatRun,
    finishChatRun,
    interruptChatRun,
    awaitUser,
    answerPendingAsk,
    touchChat,
    incrementChatTotalTokens,
    addMessage,
    removeMessagesByRunId,
    appendToolCall,
    clearError,
    cancel,
    saveWindowState,
    loadWindowState,
  };
});
