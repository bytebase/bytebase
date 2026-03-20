import { defineStore } from "pinia";
import { v4 as uuidv4 } from "uuid";
import { computed, ref, watch } from "vue";
import type {
  AgentMessage,
  AgentThread,
  AgentThreadSnapshot,
  AgentThreadStatus,
  Message,
  ToolCall,
} from "../logic/types";

export const AGENT_STATE_KEY = "bb-agent-state";
export const AGENT_WINDOW_KEY = "bb-agent-window";
export const LEGACY_AGENT_MESSAGES_KEY = "bb-agent-messages";

interface PersistedAgentState {
  currentThreadId: string | null;
  threads: AgentThread[];
  messagesByThreadId: Record<string, AgentMessage[]>;
}

interface CreateThreadOptions {
  page?: AgentThreadSnapshot;
  select?: boolean;
  title?: string;
}

interface AddMessageOptions extends Message {
  threadId?: string;
  metadata?: AgentMessage["metadata"];
}

const DEFAULT_THREAD_STATUS: AgentThreadStatus = "idle";

const createThreadRecord = (options: CreateThreadOptions = {}): AgentThread => {
  const now = Date.now();
  return {
    id: uuidv4(),
    title: options.title ?? "",
    createdTs: now,
    updatedTs: now,
    status: DEFAULT_THREAD_STATUS,
    page: options.page,
    lastError: null,
    interrupted: false,
  };
};

const getThreadTitleFromMessage = (content?: string) => {
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
  threadId: string,
  fallbackTs: number
): AgentMessage => {
  const message = isRecord(raw) ? raw : {};
  return {
    id: typeof message.id === "string" ? message.id : uuidv4(),
    threadId:
      typeof message.threadId === "string" ? message.threadId : threadId,
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

const normalizeThread = (raw: unknown): AgentThread => {
  const thread = isRecord(raw) ? raw : {};
  const now = Date.now();
  const status =
    thread.status === "running" ||
    thread.status === "awaiting_user" ||
    thread.status === "error"
      ? thread.status
      : DEFAULT_THREAD_STATUS;
  return {
    id: typeof thread.id === "string" ? thread.id : uuidv4(),
    title: typeof thread.title === "string" ? thread.title : "",
    createdTs: typeof thread.createdTs === "number" ? thread.createdTs : now,
    updatedTs: typeof thread.updatedTs === "number" ? thread.updatedTs : now,
    status,
    page:
      isRecord(thread.page) &&
      typeof thread.page.path === "string" &&
      typeof thread.page.title === "string"
        ? {
            path: thread.page.path,
            title: thread.page.title,
          }
        : undefined,
    lastError:
      typeof thread.lastError === "string"
        ? thread.lastError
        : thread.lastError === null
          ? null
          : null,
    interrupted: Boolean(thread.interrupted),
  };
};

const sortMessages = (messages: AgentMessage[]) => {
  messages.sort((a, b) => a.createdTs - b.createdTs);
  return messages;
};

const migrateLegacyState = (legacyMessages: unknown[]): PersistedAgentState => {
  const thread = createThreadRecord();
  const startTs = Date.now();
  const messages = sortMessages(
    legacyMessages.map((message, index) =>
      normalizeMessage(message, thread.id, startTs + index)
    )
  );
  const firstUserMessage = messages.find((message) => message.role === "user");
  const lastMessage = messages.at(-1);
  thread.title = getThreadTitleFromMessage(firstUserMessage?.content);
  if (lastMessage) {
    thread.updatedTs = lastMessage.createdTs;
  }
  return {
    currentThreadId: thread.id,
    threads: [thread],
    messagesByThreadId: {
      [thread.id]: messages,
    },
  };
};

const normalizePersistedState = (raw: unknown): PersistedAgentState => {
  if (!isRecord(raw)) {
    return {
      currentThreadId: null,
      threads: [],
      messagesByThreadId: {},
    };
  }

  const rawThreads = Array.isArray(raw.threads) ? raw.threads : [];
  const rawMessagesByThreadId = isRecord(raw.messagesByThreadId)
    ? raw.messagesByThreadId
    : {};

  const threads = rawThreads.map((thread) => normalizeThread(thread));
  const messagesByThreadId: Record<string, AgentMessage[]> = {};

  for (const thread of threads) {
    const rawMessages: unknown[] = Array.isArray(rawMessagesByThreadId[thread.id])
      ? (rawMessagesByThreadId[thread.id] as unknown[])
      : [];
    const messages = sortMessages(
      rawMessages.map((message, index) =>
        normalizeMessage(message, thread.id, thread.createdTs + index)
      )
    );

    const firstUserMessage = messages.find(
      (message) => message.role === "user"
    );
    const lastMessage = messages.at(-1);

    if (!thread.title) {
      thread.title = getThreadTitleFromMessage(firstUserMessage?.content);
    }
    if (lastMessage) {
      thread.updatedTs = Math.max(thread.updatedTs, lastMessage.createdTs);
    }
    if (thread.status === "running") {
      thread.status = "error";
      thread.interrupted = true;
      thread.lastError = null;
    }

    messagesByThreadId[thread.id] = messages;
  }

  const currentThreadId =
    typeof raw.currentThreadId === "string" &&
    threads.some((thread) => thread.id === raw.currentThreadId)
      ? raw.currentThreadId
      : (threads[0]?.id ?? null);

  return {
    currentThreadId,
    threads,
    messagesByThreadId,
  };
};

export const useAgentStore = defineStore("agent", () => {
  const visible = ref(false);
  const position = ref({
    x: window.innerWidth - 420,
    y: window.innerHeight - 520,
  });
  const size = ref({ width: 400, height: 500 });
  const minimized = ref(false);

  const threads = ref<AgentThread[]>([]);
  const messagesByThreadId = ref<Record<string, AgentMessage[]>>({});
  const currentThreadId = ref<string | null>(null);
  const abortController = ref<AbortController | null>(null);

  const orderedThreads = computed(() =>
    [...threads.value].sort((a, b) => b.updatedTs - a.updatedTs)
  );
  const currentThread = computed(
    () =>
      threads.value.find((thread) => thread.id === currentThreadId.value) ??
      null
  );
  const messages = computed(
    () =>
      (currentThreadId.value
        ? messagesByThreadId.value[currentThreadId.value]
        : undefined) ?? []
  );
  const runningThread = computed(
    () => threads.value.find((thread) => thread.status === "running") ?? null
  );
  const loading = computed(() => currentThread.value?.status === "running");
  const error = computed(() => currentThread.value?.lastError ?? null);
  const hasRunningThread = computed(() => !!runningThread.value);
  const runningThreadId = computed(() => runningThread.value?.id ?? null);

  const getThread = (threadId?: string | null) => {
    if (!threadId) {
      return null;
    }
    return threads.value.find((thread) => thread.id === threadId) ?? null;
  };

  const getMessages = (threadId?: string | null) => {
    if (!threadId) {
      return [];
    }
    return messagesByThreadId.value[threadId] ?? [];
  };

  const touchThread = (threadId: string) => {
    const thread = getThread(threadId);
    if (!thread) {
      return null;
    }
    thread.updatedTs = Date.now();
    return thread;
  };

  const setThreadStatus = (
    threadId: string,
    status: AgentThreadStatus,
    options: {
      interrupted?: boolean;
      lastError?: string | null;
      page?: AgentThreadSnapshot;
    } = {}
  ) => {
    const thread = getThread(threadId);
    if (!thread) {
      return null;
    }
    thread.status = status;
    thread.interrupted = options.interrupted ?? false;
    thread.lastError = options.lastError ?? null;
    if (options.page) {
      thread.page = options.page;
    }
    touchThread(threadId);
    return thread;
  };

  const createThread = (options: CreateThreadOptions = {}) => {
    const thread = createThreadRecord(options);
    threads.value.push(thread);
    messagesByThreadId.value[thread.id] = [];
    if (options.select ?? true) {
      currentThreadId.value = thread.id;
    }
    return thread;
  };

  const ensureCurrentThread = (page?: AgentThreadSnapshot) => {
    const existing = getThread(currentThreadId.value);
    if (existing) {
      if (page) {
        existing.page = page;
        touchThread(existing.id);
      }
      return existing;
    }
    return createThread({ page });
  };

  const setCurrentThread = (threadId: string) => {
    if (getThread(threadId)) {
      currentThreadId.value = threadId;
    }
  };

  const updateThreadPage = (threadId: string, page: AgentThreadSnapshot) => {
    const thread = getThread(threadId);
    if (!thread) {
      return null;
    }
    thread.page = page;
    touchThread(threadId);
    return thread;
  };

  const addMessage = (message: AddMessageOptions) => {
    const thread = getThread(message.threadId) ?? ensureCurrentThread();
    const createdTs = Date.now();
    const agentMessage: AgentMessage = {
      ...message,
      id: uuidv4(),
      threadId: thread.id,
      createdTs,
    };
    const threadMessages = getMessages(thread.id);
    if (threadMessages.length === 0) {
      messagesByThreadId.value[thread.id] = threadMessages;
    }
    threadMessages.push(agentMessage);
    if (message.role === "user" && !thread.title) {
      thread.title = getThreadTitleFromMessage(message.content);
    }
    thread.lastError = null;
    thread.interrupted = false;
    thread.updatedTs = createdTs;
    return agentMessage;
  };

  const appendToolCall = (
    threadId: string,
    messageId: string,
    toolCall: ToolCall
  ) => {
    const message = getMessages(threadId).find(
      (message) => message.id === messageId
    );
    if (!message) {
      return null;
    }
    message.toolCalls = [...(message.toolCalls ?? []), toolCall];
    touchThread(threadId);
    return message;
  };

  const clearMessages = (threadId = currentThreadId.value) => {
    const thread = getThread(threadId);
    if (!thread) {
      return;
    }
    messagesByThreadId.value[thread.id] = [];
    thread.title = "";
    thread.status = DEFAULT_THREAD_STATUS;
    thread.lastError = null;
    thread.interrupted = false;
    thread.updatedTs = Date.now();
  };

  const clearConversation = (threadId = currentThreadId.value) => {
    if (threadId && runningThreadId.value === threadId) {
      cancel();
    }
    clearMessages(threadId);
  };

  const clearError = (threadId = currentThreadId.value) => {
    const thread = getThread(threadId);
    if (!thread) {
      return;
    }
    thread.lastError = null;
    thread.interrupted = false;
    if (thread.status === "error") {
      thread.status = DEFAULT_THREAD_STATUS;
    }
    touchThread(thread.id);
  };

  const cancel = () => {
    abortController.value?.abort();
    abortController.value = null;
    if (runningThreadId.value) {
      setThreadStatus(runningThreadId.value, DEFAULT_THREAD_STATUS);
    }
  };

  const startRun = (threadId: string, page?: AgentThreadSnapshot) => {
    setThreadStatus(threadId, "running", { page });
  };

  const finishRun = (
    threadId: string,
    options: {
      status?: Extract<AgentThreadStatus, "idle" | "error">;
      lastError?: string | null;
    } = {}
  ) => {
    setThreadStatus(threadId, options.status ?? DEFAULT_THREAD_STATUS, {
      lastError: options.lastError ?? null,
      interrupted: false,
    });
  };

  const saveWindowState = () => {
    localStorage.setItem(
      AGENT_WINDOW_KEY,
      JSON.stringify({ position: position.value, size: size.value })
    );
  };

  const saveState = () => {
    const persistedState: PersistedAgentState = {
      currentThreadId: currentThreadId.value,
      threads: threads.value,
      messagesByThreadId: messagesByThreadId.value,
    };
    localStorage.setItem(AGENT_STATE_KEY, JSON.stringify(persistedState));
  };

  const loadState = () => {
    const saved = localStorage.getItem(AGENT_STATE_KEY);
    if (saved) {
      try {
        const state = normalizePersistedState(JSON.parse(saved));
        threads.value = state.threads;
        messagesByThreadId.value = state.messagesByThreadId;
        currentThreadId.value = state.currentThreadId;
      } catch {
        localStorage.removeItem(AGENT_STATE_KEY);
      }
    } else {
      const legacySaved = localStorage.getItem(LEGACY_AGENT_MESSAGES_KEY);
      if (legacySaved) {
        try {
          const parsed = JSON.parse(legacySaved);
          if (Array.isArray(parsed)) {
            const state = migrateLegacyState(parsed);
            threads.value = state.threads;
            messagesByThreadId.value = state.messagesByThreadId;
            currentThreadId.value = state.currentThreadId;
          }
        } catch {
          localStorage.removeItem(LEGACY_AGENT_MESSAGES_KEY);
        }
        localStorage.removeItem(LEGACY_AGENT_MESSAGES_KEY);
      }
    }

    if (!currentThreadId.value || !getThread(currentThreadId.value)) {
      const thread = createThread();
      currentThreadId.value = thread.id;
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
      };
      if (state.position?.x && state.position?.y) {
        position.value = {
          x: state.position.x,
          y: state.position.y,
        };
      }
      if (state.size?.width && state.size?.height) {
        size.value = {
          width: state.size.width,
          height: state.size.height,
        };
      }
    } catch {
      localStorage.removeItem(AGENT_WINDOW_KEY);
    }
  };

  watch([threads, messagesByThreadId, currentThreadId], saveState, {
    deep: true,
  });

  loadState();

  return {
    visible,
    position,
    size,
    minimized,
    threads,
    orderedThreads,
    currentThreadId,
    currentThread,
    messages,
    loading,
    error,
    hasRunningThread,
    runningThreadId,
    abortController,
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
    getThread,
    getMessages,
    createThread,
    ensureCurrentThread,
    setCurrentThread,
    updateThreadPage,
    setThreadStatus,
    startRun,
    finishRun,
    touchThread,
    addMessage,
    appendToolCall,
    clearMessages,
    clearConversation,
    clearError,
    cancel,
    saveWindowState,
    loadWindowState,
  };
});
