import { v4 as uuidv4 } from "uuid";
import { create } from "zustand";
import { immer } from "zustand/middleware/immer";
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

// --- Pure helper functions (copied verbatim from Pinia store) ---

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
    requiresAIConfiguration: false,
    interrupted: false,
    runId: null,
  };
};

const normalizeChatTitle = (title?: string) => {
  return (title ?? "").trim();
};

const getChatTitleFromMessage = (content?: string) => {
  const title = normalizeChatTitle(content).replace(/\s+/g, " ");
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
    requiresAIConfiguration: Boolean(chat.requiresAIConfiguration),
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
      chat.requiresAIConfiguration = false;
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

// --- Zustand store types ---

export interface AgentState {
  // Window state
  visible: boolean;
  position: { x: number; y: number };
  size: { width: number; height: number };
  sidebarWidth: number;
  minimized: boolean;

  // Chat state
  chats: AgentChat[];
  messagesByChatId: Record<string, AgentMessage[]>;
  pendingAskByChatId: Record<string, AgentPendingAsk>;
  currentChatId: string | null;
  abortControllersByChatId: Record<string, AbortController>;

  // UI actions
  toggle: () => void;
  minimize: () => void;
  restore: () => void;
  setPosition: (x: number, y: number) => void;
  setSize: (width: number, height: number) => void;
  setSidebarWidth: (width: number) => void;

  // Chat CRUD
  createChat: (options?: CreateChatOptions) => AgentChat;
  ensureCurrentChat: (page?: AgentChatSnapshot) => AgentChat;
  setCurrentChat: (chatId: string) => void;
  updateChatPage: (chatId: string, page: AgentChatSnapshot) => AgentChat | null;
  renameChat: (chatId: string, title: string) => AgentChat | null;
  archiveChat: (chatId: string) => AgentChat | null;
  unarchiveChat: (chatId: string) => AgentChat | null;
  deleteChat: (chatId: string) => boolean;

  // Messages
  addMessage: (message: AddMessageOptions) => AgentMessage;
  removeMessagesByRunId: (
    chatId: string,
    runId?: string | null
  ) => AgentMessage[];
  appendToolCall: (
    chatId: string,
    messageId: string,
    toolCall: ToolCall
  ) => AgentMessage | null;

  // Status
  setChatStatus: (
    chatId: string,
    status: AgentChatStatus,
    options?: {
      interrupted?: boolean;
      lastError?: string | null;
      requiresAIConfiguration?: boolean;
      page?: AgentChatSnapshot;
    }
  ) => AgentChat | null;
  startChatRun: (
    chatId: string,
    page?: AgentChatSnapshot,
    options?: { runId?: string }
  ) => void;
  finishChatRun: (
    chatId: string,
    options?: {
      status?: Extract<AgentChatStatus, "idle" | "error">;
      lastError?: string | null;
      requiresAIConfiguration?: boolean;
    }
  ) => void;
  interruptChatRun: (chatId: string, page?: AgentChatSnapshot) => void;

  // Pending ask
  setPendingAsk: (
    chatId: string,
    pendingAsk: AgentPendingAsk
  ) => AgentPendingAsk;
  clearPendingAsk: (chatId?: string | null) => void;
  awaitUser: (chatId: string, pendingAsk: AgentPendingAsk) => AgentPendingAsk;
  answerPendingAsk: (
    chatId: string,
    response: AgentAskUserResponse,
    metadata?: AgentMessage["metadata"]
  ) => AgentMessage | null;

  // Token
  incrementChatTotalTokens: (
    chatId: string,
    totalTokensUsed: number
  ) => AgentChat | null;

  // Abort
  setAbortController: (
    chatId: string,
    controller: AbortController | null
  ) => AbortController | null;
  getAbortController: (chatId?: string | null) => AbortController | null;
  cancel: (chatId?: string | null) => void;

  // Error
  clearError: (chatId?: string | null) => void;

  // Persistence
  saveWindowState: () => void;
  loadWindowState: () => void;
  saveState: () => void;
  loadState: () => void;

  // Queries
  getChat: (chatId?: string | null) => AgentChat | null;
  getMessages: (chatId?: string | null) => AgentMessage[];
  getPendingAsk: (chatId?: string | null) => AgentPendingAsk | null;
  isChatRunning: (chatId?: string | null) => boolean;
  canSelectChat: (chatId?: string | null) => boolean;
  touchChat: (chatId: string) => AgentChat | null;
}

// --- Selector functions (derived values) ---

// Stable empty references — avoids creating new objects on every selector call,
// which would cause infinite re-render loops with useSyncExternalStore.
const EMPTY_MESSAGES: AgentMessage[] = [];
const EMPTY_CHATS: AgentChat[] = [];

// Memoised selector: only re-sorts when the chats array reference changes.
let _orderedChatsInput: AgentChat[] | null = null;
let _orderedChatsResult: AgentChat[] = EMPTY_CHATS;

export const selectOrderedChats = (state: AgentState): AgentChat[] => {
  if (state.chats !== _orderedChatsInput) {
    _orderedChatsInput = state.chats;
    _orderedChatsResult = [...state.chats].sort((a, b) => {
      if (b.updatedTs !== a.updatedTs) {
        return b.updatedTs - a.updatedTs;
      }
      return b.createdTs - a.createdTs;
    });
  }
  return _orderedChatsResult;
};

export const selectCurrentChat = (state: AgentState): AgentChat | null =>
  state.chats.find((chat) => chat.id === state.currentChatId) ?? null;

export const selectMessages = (state: AgentState): AgentMessage[] =>
  (state.currentChatId
    ? state.messagesByChatId[state.currentChatId]
    : undefined) ?? EMPTY_MESSAGES;

export const selectCurrentPendingAsk = (
  state: AgentState
): AgentPendingAsk | null =>
  (state.currentChatId
    ? state.pendingAskByChatId[state.currentChatId]
    : undefined) ?? null;

export const selectLoading = (state: AgentState): boolean =>
  selectCurrentChat(state)?.status === "running";

export const selectError = (state: AgentState): string | null =>
  selectCurrentChat(state)?.lastError ?? null;

export const selectCurrentChatRequiresAIConfiguration = (
  state: AgentState
): boolean => selectCurrentChat(state)?.requiresAIConfiguration ?? false;

export const selectHasRunningChat = (state: AgentState): boolean =>
  state.chats.some((chat) => chat.status === "running");

// --- Store creation ---

const getSafeStorage = (): Storage | null => {
  const storage = globalThis.localStorage as Partial<Storage> | undefined;
  if (
    !storage ||
    typeof storage.getItem !== "function" ||
    typeof storage.setItem !== "function" ||
    typeof storage.removeItem !== "function"
  ) {
    return null;
  }
  return storage as Storage;
};

const loadInitialState = (): {
  chats: AgentChat[];
  messagesByChatId: Record<string, AgentMessage[]>;
  pendingAskByChatId: Record<string, AgentPendingAsk>;
  currentChatId: string | null;
} => {
  const storage = getSafeStorage();
  const saved = storage?.getItem(AGENT_STATE_KEY);
  let state: PersistedAgentState | null = null;

  if (saved) {
    try {
      state = normalizePersistedState(JSON.parse(saved));
    } catch {
      storage?.removeItem(AGENT_STATE_KEY);
    }
  }

  if (
    state &&
    state.currentChatId &&
    state.chats.some((c) => c.id === state!.currentChatId)
  ) {
    return state;
  }

  // Need a default chat
  const chats = state?.chats ?? [];
  const messagesByChatId = state?.messagesByChatId ?? {};
  const pendingAskByChatId = state?.pendingAskByChatId ?? {};
  const defaultChat = createChatRecord();
  chats.push(defaultChat);
  messagesByChatId[defaultChat.id] = [];

  return {
    chats,
    messagesByChatId,
    pendingAskByChatId,
    currentChatId: defaultChat.id,
  };
};

export const createAgentStore = () => {
  const initial = loadInitialState();

  const store = create<AgentState>()(
    immer((set, get) => {
      // Internal helpers that work on the plain (non-draft) state via get()
      const _getChat = (chatId?: string | null): AgentChat | null => {
        if (!chatId) return null;
        return get().chats.find((chat) => chat.id === chatId) ?? null;
      };

      const _hasRunningChat = (): boolean =>
        get().chats.some((chat) => chat.status === "running");

      const _touchChatInDraft = (
        chats: AgentChat[],
        chatId: string
      ): AgentChat | null => {
        const chat = chats.find((c) => c.id === chatId);
        if (!chat) return null;
        chat.updatedTs = Date.now();
        return chat;
      };

      return {
        // Window state
        visible: false,
        position: {
          x: typeof window !== "undefined" ? window.innerWidth - 420 : 0,
          y: typeof window !== "undefined" ? window.innerHeight - 520 : 0,
        },
        size: { width: 400, height: 500 },
        sidebarWidth: 256,
        minimized: false,

        // Chat state (from persisted/initial)
        chats: initial.chats,
        messagesByChatId: initial.messagesByChatId,
        pendingAskByChatId: initial.pendingAskByChatId,
        currentChatId: initial.currentChatId,
        abortControllersByChatId: {},

        // UI actions
        toggle: () =>
          set((state) => {
            state.visible = !state.visible;
            if (state.visible) {
              state.minimized = false;
            }
          }),
        minimize: () =>
          set((state) => {
            state.minimized = true;
          }),
        restore: () =>
          set((state) => {
            state.minimized = false;
          }),
        setPosition: (x, y) =>
          set((state) => {
            state.position = { x, y };
          }),
        setSize: (width, height) =>
          set((state) => {
            state.size = { width, height };
          }),
        setSidebarWidth: (width) =>
          set((state) => {
            state.sidebarWidth = width;
          }),

        // Queries
        getChat: (chatId) => _getChat(chatId),

        getMessages: (chatId) => {
          if (!chatId) return [];
          return get().messagesByChatId[chatId] ?? [];
        },

        getPendingAsk: (chatId) => {
          const id = chatId ?? get().currentChatId;
          if (!id) return null;
          return get().pendingAskByChatId[id] ?? null;
        },

        getAbortController: (chatId) => {
          if (!chatId) return null;
          return get().abortControllersByChatId[chatId] ?? null;
        },

        isChatRunning: (chatId) => _getChat(chatId)?.status === "running",

        canSelectChat: (chatId) => {
          if (!chatId || !_getChat(chatId)) return false;
          return !_hasRunningChat() || get().currentChatId === chatId;
        },

        touchChat: (chatId) => {
          let result: AgentChat | null = null;
          set((state) => {
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              chat.updatedTs = Date.now();
              result = { ...chat } as AgentChat;
            }
          });
          return result;
        },

        setAbortController: (chatId, controller) => {
          // AbortControllers are not serializable, store outside immer
          const controllers = get().abortControllersByChatId;
          if (controller) {
            set((state) => {
              state.abortControllersByChatId[chatId] = controller;
            });
            return controller;
          }
          if (controllers[chatId]) {
            set((state) => {
              delete state.abortControllersByChatId[chatId];
            });
          }
          return null;
        },

        // Chat CRUD
        createChat: (options = {}) => {
          const chat = createChatRecord(options);
          const shouldSelect =
            (options.select ?? true) &&
            (!_hasRunningChat() || get().currentChatId === null);
          set((state) => {
            state.chats.push(chat);
            state.messagesByChatId[chat.id] = [];
            if (shouldSelect) {
              state.currentChatId = chat.id;
            }
          });
          return chat;
        },

        ensureCurrentChat: (page) => {
          const existing = _getChat(get().currentChatId);
          if (existing) {
            if (page && !existing.page) {
              set((state) => {
                const chat = state.chats.find((c) => c.id === existing.id);
                if (chat) {
                  chat.page = page;
                  _touchChatInDraft(state.chats, chat.id);
                }
              });
            }
            return _getChat(existing.id)!;
          }
          return get().createChat({ page });
        },

        setCurrentChat: (chatId) => {
          if (get().canSelectChat(chatId)) {
            set((state) => {
              state.currentChatId = chatId;
            });
          }
        },

        updateChatPage: (chatId, page) => {
          if (!_getChat(chatId)) return null;
          set((state) => {
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              chat.page = page;
              _touchChatInDraft(state.chats, chatId);
            }
          });
          return _getChat(chatId);
        },

        renameChat: (chatId, title) => {
          const chat = _getChat(chatId);
          if (!chat) return null;
          const normalizedTitle = normalizeChatTitle(title);
          if (chat.title === normalizedTitle) return chat;
          set((state) => {
            const c = state.chats.find((c) => c.id === chatId);
            if (c) {
              c.title = normalizedTitle;
              _touchChatInDraft(state.chats, chatId);
            }
          });
          return _getChat(chatId);
        },

        archiveChat: (chatId) => {
          if (!_getChat(chatId)) return null;
          set((state) => {
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              chat.archived = true;
              _touchChatInDraft(state.chats, chatId);
            }
          });
          return _getChat(chatId);
        },

        unarchiveChat: (chatId) => {
          if (!_getChat(chatId)) return null;
          set((state) => {
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              chat.archived = false;
              _touchChatInDraft(state.chats, chatId);
            }
          });
          return _getChat(chatId);
        },

        deleteChat: (chatId) => {
          if (!_getChat(chatId)) return false;
          get().cancel(chatId);
          set((state) => {
            state.chats = state.chats.filter((c) => c.id !== chatId);
            delete state.messagesByChatId[chatId];
            delete state.pendingAskByChatId[chatId];
            delete state.abortControllersByChatId[chatId];
            if (state.currentChatId === chatId) {
              // Select next available chat
              const ordered = [...state.chats].sort((a, b) => {
                if (b.updatedTs !== a.updatedTs)
                  return b.updatedTs - a.updatedTs;
                return b.createdTs - a.createdTs;
              });
              if (ordered.length > 0) {
                state.currentChatId = ordered[0].id;
              } else {
                // Create a new default chat
                const newChat = createChatRecord();
                state.chats.push(newChat);
                state.messagesByChatId[newChat.id] = [];
                state.currentChatId = newChat.id;
              }
            }
          });
          return true;
        },

        // Messages
        addMessage: (message) => {
          const chatId =
            message.chatId && _getChat(message.chatId)
              ? message.chatId
              : get().ensureCurrentChat().id;
          const createdTs = Date.now();
          const agentMessage: AgentMessage = {
            ...message,
            id: uuidv4(),
            chatId,
            createdTs,
          };
          set((state) => {
            const chatMessages = state.messagesByChatId[chatId] ?? [];
            if (!state.messagesByChatId[chatId]) {
              state.messagesByChatId[chatId] = chatMessages;
            }
            chatMessages.push(agentMessage);
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              if (message.role === "user" && !chat.title) {
                chat.title = getChatTitleFromMessage(message.content);
              }
              chat.lastError = null;
              chat.requiresAIConfiguration = false;
              chat.interrupted = false;
              chat.updatedTs = createdTs;
            }
          });
          return agentMessage;
        },

        removeMessagesByRunId: (chatId, runId) => {
          if (!runId || !_getChat(chatId)) return [];
          const existingMessages = get().messagesByChatId[chatId] ?? [];
          const removedMessages = existingMessages.filter(
            (m) => m.metadata?.runId === runId
          );
          if (removedMessages.length === 0) return [];
          set((state) => {
            state.messagesByChatId[chatId] = (
              state.messagesByChatId[chatId] ?? []
            ).filter((m) => m.metadata?.runId !== runId);
            _touchChatInDraft(state.chats, chatId);
          });
          return removedMessages;
        },

        appendToolCall: (chatId, messageId, toolCall) => {
          const messages = get().messagesByChatId[chatId] ?? [];
          const message = messages.find((m) => m.id === messageId);
          if (!message) return null;
          set((state) => {
            const msgs = state.messagesByChatId[chatId] ?? [];
            const msg = msgs.find((m) => m.id === messageId);
            if (msg) {
              msg.toolCalls = [...(msg.toolCalls ?? []), toolCall];
              _touchChatInDraft(state.chats, chatId);
            }
          });
          return (
            get()
              .getMessages(chatId)
              .find((m) => m.id === messageId) ?? null
          );
        },

        // Status
        setChatStatus: (chatId, status, options = {}) => {
          if (!_getChat(chatId)) return null;
          set((state) => {
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              chat.status = status;
              chat.interrupted = options.interrupted ?? false;
              chat.lastError = options.lastError ?? null;
              chat.requiresAIConfiguration =
                options.requiresAIConfiguration ?? false;
              if (options.page) {
                chat.page = options.page;
              }
              if (status !== "awaiting_user") {
                delete state.pendingAskByChatId[chatId];
              }
              _touchChatInDraft(state.chats, chatId);
            }
          });
          return _getChat(chatId);
        },

        startChatRun: (chatId, page, options = {}) => {
          get().setChatStatus(chatId, "running", { page });
          set((state) => {
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              chat.runId = options.runId ?? chat.runId ?? null;
            }
          });
        },

        finishChatRun: (chatId, options = {}) => {
          get().setChatStatus(chatId, options.status ?? DEFAULT_CHAT_STATUS, {
            lastError: options.lastError ?? null,
            requiresAIConfiguration: options.requiresAIConfiguration ?? false,
            interrupted: false,
          });
          set((state) => {
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              chat.runId = null;
            }
          });
        },

        interruptChatRun: (chatId, page) => {
          get().setChatStatus(chatId, DEFAULT_CHAT_STATUS, {
            interrupted: true,
            page,
          });
        },

        // Pending ask
        setPendingAsk: (chatId, pendingAsk) => {
          set((state) => {
            state.pendingAskByChatId[chatId] = pendingAsk;
            _touchChatInDraft(state.chats, chatId);
          });
          return pendingAsk;
        },

        clearPendingAsk: (chatId) => {
          const id = chatId ?? get().currentChatId;
          if (!id || !get().pendingAskByChatId[id]) return;
          set((state) => {
            delete state.pendingAskByChatId[id];
            _touchChatInDraft(state.chats, id);
          });
        },

        awaitUser: (chatId, pendingAsk) => {
          get().setChatStatus(chatId, "awaiting_user");
          return get().setPendingAsk(chatId, pendingAsk);
        },

        answerPendingAsk: (chatId, response, metadata) => {
          const pendingAsk = get().getPendingAsk(chatId);
          if (!pendingAsk) return null;
          const toolMessage = get().addMessage({
            chatId,
            role: "tool",
            toolCallId: pendingAsk.toolCallId,
            content: JSON.stringify(response),
            metadata,
          });
          get().clearPendingAsk(chatId);
          return toolMessage;
        },

        // Token
        incrementChatTotalTokens: (chatId, totalTokensUsed) => {
          if (!_getChat(chatId) || totalTokensUsed <= 0) return null;
          set((state) => {
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              chat.totalTokensUsed += totalTokensUsed;
              _touchChatInDraft(state.chats, chatId);
            }
          });
          return _getChat(chatId);
        },

        // Abort / cancel
        cancel: (chatId) => {
          const id = chatId ?? get().currentChatId;
          if (!id) return;
          get().getAbortController(id)?.abort();
          get().setAbortController(id, null);
          if (get().isChatRunning(id)) {
            get().interruptChatRun(id);
          }
        },

        // Error
        clearError: (chatId) => {
          const id = chatId ?? get().currentChatId;
          const chat = _getChat(id);
          if (!chat) return;
          set((state) => {
            const c = state.chats.find((c) => c.id === id);
            if (c) {
              c.lastError = null;
              c.requiresAIConfiguration = false;
              c.interrupted = false;
              c.runId = null;
              if (c.status === "error") {
                c.status = DEFAULT_CHAT_STATUS;
              }
              _touchChatInDraft(state.chats, c.id);
            }
          });
        },

        // Persistence
        saveWindowState: () => {
          const storage = getSafeStorage();
          if (!storage) return;
          const state = get();
          storage.setItem(
            AGENT_WINDOW_KEY,
            JSON.stringify({
              position: state.position,
              size: state.size,
              sidebarWidth: state.sidebarWidth,
            })
          );
        },

        loadWindowState: () => {
          const storage = getSafeStorage();
          const saved = storage?.getItem(AGENT_WINDOW_KEY);
          if (!saved) return;
          try {
            const state = JSON.parse(saved) as {
              position?: { x?: number; y?: number };
              size?: { width?: number; height?: number };
              sidebarWidth?: number;
            };
            set((draft) => {
              if (
                typeof state.position?.x === "number" &&
                typeof state.position?.y === "number"
              ) {
                draft.position = {
                  x: state.position.x,
                  y: state.position.y,
                };
              }
              if (
                typeof state.size?.width === "number" &&
                typeof state.size?.height === "number"
              ) {
                draft.size = {
                  width: state.size.width,
                  height: state.size.height,
                };
              }
              if (typeof state.sidebarWidth === "number") {
                draft.sidebarWidth = state.sidebarWidth;
              }
            });
          } catch {
            storage?.removeItem(AGENT_WINDOW_KEY);
          }
        },

        saveState: () => {
          const storage = getSafeStorage();
          if (!storage) return;
          const state = get();
          const persistedState: PersistedAgentState = {
            currentChatId: state.currentChatId,
            chats: state.chats,
            messagesByChatId: state.messagesByChatId,
            pendingAskByChatId: state.pendingAskByChatId,
          };
          storage.setItem(AGENT_STATE_KEY, JSON.stringify(persistedState));
        },

        loadState: () => {
          const storage = getSafeStorage();
          const saved = storage?.getItem(AGENT_STATE_KEY);
          if (saved) {
            try {
              const state = normalizePersistedState(JSON.parse(saved));
              set((draft) => {
                draft.chats = state.chats;
                draft.messagesByChatId = state.messagesByChatId;
                draft.pendingAskByChatId = state.pendingAskByChatId;
                draft.currentChatId = state.currentChatId;
              });
            } catch {
              storage?.removeItem(AGENT_STATE_KEY);
            }
          }

          const s = get();
          if (
            !s.currentChatId ||
            !s.chats.some((c) => c.id === s.currentChatId)
          ) {
            const chat = createChatRecord();
            set((draft) => {
              draft.chats.push(chat);
              draft.messagesByChatId[chat.id] = [];
              draft.currentChatId = chat.id;
            });
          }
        },
      };
    })
  );

  // Auto-persist on state changes
  // Auto-persist only the chat state slice (not UI/window state).
  // Uses a serialization check to avoid redundant writes during
  // drag/resize operations that only change position/size.
  let lastSerialized = "";
  store.subscribe((state) => {
    const storage = getSafeStorage();
    if (!storage) {
      return;
    }
    const persistedState: PersistedAgentState = {
      currentChatId: state.currentChatId,
      chats: state.chats,
      messagesByChatId: state.messagesByChatId,
      pendingAskByChatId: state.pendingAskByChatId,
    };
    const serialized = JSON.stringify(persistedState);
    if (serialized !== lastSerialized) {
      lastSerialized = serialized;
      storage.setItem(AGENT_STATE_KEY, serialized);
    }
  });

  // Load persisted window state (position, size, sidebar width) on creation.
  store.getState().loadWindowState();

  return store;
};

// Singleton for production use.
// The return value of create() IS a React hook — callable as
// useAgentStore(selector) to subscribe React components to state slices.
// Also callable outside React via useAgentStore.getState().
export const useAgentStore = createAgentStore();
