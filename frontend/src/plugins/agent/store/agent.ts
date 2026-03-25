import { defineStore } from "pinia";
import { v4 as uuidv4 } from "uuid";
import { computed, ref, watch } from "vue";
import type {
  AgentAskUserResponse,
  AgentMessage,
  AgentPendingAsk,
  AgentThread,
  AgentThreadSnapshot,
  AgentThreadStatus,
  Message,
  ToolCall,
} from "../logic/types";

export const AGENT_STATE_KEY = "bb-agent-state-v2";
export const AGENT_WINDOW_KEY = "bb-agent-window";

interface PersistedAgentState {
  currentThreadId: string | null;
  threads: AgentThread[];
  messagesByThreadId: Record<string, AgentMessage[]>;
  pendingAskByThreadId: Record<string, AgentPendingAsk>;
}

interface CreateThreadOptions {
  page?: AgentThreadSnapshot;
  select?: boolean;
  title?: string;
  archived?: boolean;
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
    totalTokensUsed: 0,
    page: options.page,
    archived: options.archived ?? false,
    lastError: null,
    interrupted: false,
    runId: null,
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
    totalTokensUsed:
      typeof thread.totalTokensUsed === "number" &&
      Number.isFinite(thread.totalTokensUsed) &&
      thread.totalTokensUsed >= 0
        ? thread.totalTokensUsed
        : 0,
    page:
      isRecord(thread.page) &&
      typeof thread.page.path === "string" &&
      typeof thread.page.title === "string"
        ? {
            path: thread.page.path,
            title: thread.page.title,
          }
        : undefined,
    archived: Boolean(thread.archived),
    lastError:
      typeof thread.lastError === "string"
        ? thread.lastError
        : thread.lastError === null
          ? null
          : null,
    interrupted: Boolean(thread.interrupted),
    runId: typeof thread.runId === "string" ? thread.runId : null,
  };
};

const sortMessages = (messages: AgentMessage[]) => {
  messages.sort((a, b) => a.createdTs - b.createdTs);
  return messages;
};

const createEmptyPersistedState = (): PersistedAgentState => ({
  currentThreadId: null,
  threads: [],
  messagesByThreadId: {},
  pendingAskByThreadId: {},
});

const normalizePersistedState = (raw: unknown): PersistedAgentState => {
  if (!isRecord(raw)) {
    return createEmptyPersistedState();
  }

  const rawThreads = Array.isArray(raw.threads) ? raw.threads : [];
  const rawMessagesByThreadId = isRecord(raw.messagesByThreadId)
    ? raw.messagesByThreadId
    : {};
  const rawPendingAskByThreadId = isRecord(raw.pendingAskByThreadId)
    ? raw.pendingAskByThreadId
    : {};

  const threads = rawThreads.map((thread) => normalizeThread(thread));
  const messagesByThreadId: Record<string, AgentMessage[]> = {};
  const pendingAskByThreadId: Record<string, AgentPendingAsk> = {};

  for (const thread of threads) {
    const rawMessages: unknown[] = Array.isArray(
      rawMessagesByThreadId[thread.id]
    )
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
      thread.status = DEFAULT_THREAD_STATUS;
      thread.interrupted = true;
      thread.lastError = null;
    }

    const pendingAsk = normalizePendingAsk(rawPendingAskByThreadId[thread.id]);
    if (thread.status === "awaiting_user") {
      if (pendingAsk) {
        pendingAskByThreadId[thread.id] = pendingAsk;
      } else {
        thread.status = DEFAULT_THREAD_STATUS;
      }
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
    pendingAskByThreadId,
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
  const pendingAskByThreadId = ref<Record<string, AgentPendingAsk>>({});
  const currentThreadId = ref<string | null>(null);
  const abortControllersByThreadId = ref<Record<string, AbortController>>({});

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
  const currentPendingAsk = computed(
    () =>
      (currentThreadId.value
        ? pendingAskByThreadId.value[currentThreadId.value]
        : undefined) ?? null
  );
  const loading = computed(() => currentThread.value?.status === "running");
  const error = computed(() => currentThread.value?.lastError ?? null);
  const runningThreadIds = computed(() =>
    threads.value
      .filter((thread) => thread.status === "running")
      .map((thread) => thread.id)
  );
  const hasRunningThread = computed(() => runningThreadIds.value.length > 0);

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

  const getPendingAsk = (threadId = currentThreadId.value) => {
    if (!threadId) {
      return null;
    }
    return pendingAskByThreadId.value[threadId] ?? null;
  };

  const getAbortController = (threadId?: string | null) => {
    if (!threadId) {
      return null;
    }
    return abortControllersByThreadId.value[threadId] ?? null;
  };

  const isThreadRunning = (threadId?: string | null) => {
    return getThread(threadId)?.status === "running";
  };

  const touchThread = (threadId: string) => {
    const thread = getThread(threadId);
    if (!thread) {
      return null;
    }
    thread.updatedTs = Date.now();
    return thread;
  };

  const setAbortController = (
    threadId: string,
    controller: AbortController | null
  ) => {
    if (controller) {
      abortControllersByThreadId.value[threadId] = controller;
      return controller;
    }
    delete abortControllersByThreadId.value[threadId];
    return null;
  };

  const clearPendingAsk = (threadId = currentThreadId.value) => {
    if (!threadId || !pendingAskByThreadId.value[threadId]) {
      return;
    }
    delete pendingAskByThreadId.value[threadId];
    touchThread(threadId);
  };

  const setPendingAsk = (threadId: string, pendingAsk: AgentPendingAsk) => {
    pendingAskByThreadId.value[threadId] = pendingAsk;
    touchThread(threadId);
    return pendingAsk;
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
    if (status !== "awaiting_user") {
      delete pendingAskByThreadId.value[threadId];
    }
    touchThread(threadId);
    return thread;
  };

  const ensureCurrentThread = (page?: AgentThreadSnapshot) => {
    const existing = getThread(currentThreadId.value);
    if (existing) {
      if (page && !existing.page) {
        existing.page = page;
        touchThread(existing.id);
      }
      return existing;
    }
    return createThread({ page });
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

  const selectNextAvailableThread = (preferredThreadId?: string | null) => {
    const preferredThread = getThread(preferredThreadId);
    if (preferredThread) {
      currentThreadId.value = preferredThread.id;
      return preferredThread;
    }
    const fallbackThread = orderedThreads.value[0] ?? createThread();
    currentThreadId.value = fallbackThread.id;
    return fallbackThread;
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

  const renameThread = (threadId: string, title: string) => {
    const thread = getThread(threadId);
    if (!thread) {
      return null;
    }
    thread.title = title.trim();
    touchThread(threadId);
    return thread;
  };

  const archiveThread = (threadId: string) => {
    const thread = getThread(threadId);
    if (!thread) {
      return null;
    }
    thread.archived = true;
    touchThread(threadId);
    return thread;
  };

  const unarchiveThread = (threadId: string) => {
    const thread = getThread(threadId);
    if (!thread) {
      return null;
    }
    thread.archived = false;
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

  const removeMessagesByRunId = (threadId: string, runId?: string | null) => {
    if (!runId) {
      return [];
    }
    const thread = getThread(threadId);
    if (!thread) {
      return [];
    }
    const existingMessages = getMessages(threadId);
    const removedMessages = existingMessages.filter(
      (message) => message.metadata?.runId === runId
    );
    if (removedMessages.length === 0) {
      return [];
    }
    messagesByThreadId.value[threadId] = existingMessages.filter(
      (message) => message.metadata?.runId !== runId
    );
    touchThread(threadId);
    return removedMessages;
  };

  const appendToolCall = (
    threadId: string,
    messageId: string,
    toolCall: ToolCall
  ) => {
    const message = getMessages(threadId).find(
      (candidate) => candidate.id === messageId
    );
    if (!message) {
      return null;
    }
    message.toolCalls = [...(message.toolCalls ?? []), toolCall];
    touchThread(threadId);
    return message;
  };

  const incrementThreadTotalTokens = (
    threadId: string,
    totalTokensUsed: number
  ) => {
    const thread = getThread(threadId);
    if (!thread || totalTokensUsed <= 0) {
      return null;
    }
    thread.totalTokensUsed += totalTokensUsed;
    touchThread(threadId);
    return thread;
  };

  const awaitUser = (threadId: string, pendingAsk: AgentPendingAsk) => {
    setThreadStatus(threadId, "awaiting_user");
    return setPendingAsk(threadId, pendingAsk);
  };

  const answerPendingAsk = (
    threadId: string,
    response: AgentAskUserResponse,
    metadata?: AgentMessage["metadata"]
  ) => {
    const pendingAsk = getPendingAsk(threadId);
    if (!pendingAsk) {
      return null;
    }
    const toolMessage = addMessage({
      threadId,
      role: "tool",
      toolCallId: pendingAsk.toolCallId,
      content: JSON.stringify(response),
      metadata,
    });
    clearPendingAsk(threadId);
    return toolMessage;
  };

  const clearMessages = (threadId = currentThreadId.value) => {
    const thread = getThread(threadId);
    if (!thread) {
      return;
    }
    messagesByThreadId.value[thread.id] = [];
    delete pendingAskByThreadId.value[thread.id];
    thread.title = "";
    thread.page = undefined;
    thread.status = DEFAULT_THREAD_STATUS;
    thread.totalTokensUsed = 0;
    thread.lastError = null;
    thread.interrupted = false;
    thread.runId = null;
    thread.updatedTs = Date.now();
  };

  const cancel = (threadId = currentThreadId.value) => {
    if (!threadId) {
      return;
    }
    getAbortController(threadId)?.abort();
    setAbortController(threadId, null);
    if (isThreadRunning(threadId)) {
      interruptRun(threadId);
    }
  };

  const clearConversation = (threadId = currentThreadId.value) => {
    if (threadId && isThreadRunning(threadId)) {
      cancel(threadId);
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
    thread.runId = null;
    if (thread.status === "error") {
      thread.status = DEFAULT_THREAD_STATUS;
    }
    touchThread(thread.id);
  };

  const interruptRun = (threadId: string, page?: AgentThreadSnapshot) => {
    setThreadStatus(threadId, DEFAULT_THREAD_STATUS, {
      interrupted: true,
      page,
    });
  };

  const startRun = (
    threadId: string,
    page?: AgentThreadSnapshot,
    options: {
      runId?: string;
    } = {}
  ) => {
    setThreadStatus(threadId, "running", {
      page,
    });
    const thread = getThread(threadId);
    if (thread) {
      thread.runId = options.runId ?? thread.runId ?? null;
    }
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
    const thread = getThread(threadId);
    if (thread) {
      thread.runId = null;
    }
  };

  const deleteThread = (threadId: string) => {
    if (!getThread(threadId)) {
      return false;
    }
    cancel(threadId);
    threads.value = threads.value.filter((thread) => thread.id !== threadId);
    delete messagesByThreadId.value[threadId];
    delete pendingAskByThreadId.value[threadId];
    delete abortControllersByThreadId.value[threadId];
    if (currentThreadId.value === threadId) {
      selectNextAvailableThread();
    }
    return true;
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
      pendingAskByThreadId: pendingAskByThreadId.value,
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
        pendingAskByThreadId.value = state.pendingAskByThreadId;
        currentThreadId.value = state.currentThreadId;
      } catch {
        localStorage.removeItem(AGENT_STATE_KEY);
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
    } catch {
      localStorage.removeItem(AGENT_WINDOW_KEY);
    }
  };

  watch(
    [threads, messagesByThreadId, pendingAskByThreadId, currentThreadId],
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
    minimized,
    threads,
    orderedThreads,
    currentThreadId,
    currentThread,
    messages,
    currentPendingAsk,
    loading,
    error,
    runningThreadIds,
    hasRunningThread,
    abortControllersByThreadId,
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
    getPendingAsk,
    getAbortController,
    isThreadRunning,
    setAbortController,
    createThread,
    ensureCurrentThread,
    setCurrentThread,
    updateThreadPage,
    renameThread,
    archiveThread,
    unarchiveThread,
    deleteThread,
    setPendingAsk,
    clearPendingAsk,
    setThreadStatus,
    startRun,
    finishRun,
    interruptRun,
    awaitUser,
    answerPendingAsk,
    touchThread,
    incrementThreadTotalTokens,
    addMessage,
    removeMessagesByRunId,
    appendToolCall,
    clearMessages,
    clearConversation,
    clearError,
    cancel,
    saveWindowState,
    loadWindowState,
  };
});
