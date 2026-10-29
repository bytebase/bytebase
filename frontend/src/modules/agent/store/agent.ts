import { v4 as uuidv4 } from "uuid";
import { create } from "zustand";
import { immer } from "zustand/middleware/immer";
import { getResourceId, workspaceNamePrefix } from "@/lib/resourceName";
import {
  getNameParentTokens,
  projectNamePrefix,
} from "@/stores/modules/v1/common";
import type { LandingStarterId } from "../logic/landingStarters";
import {
  formatResourceMention,
  getMentionTriggerRange,
  insertResourceMention,
  isResourceMentionRange,
  normalizeAgentResourceMentions,
  normalizeAgentResourceReference,
  normalizeAgentResourceReferences,
} from "../logic/resourceMention";
import type {
  AgentAskUserResponse,
  AgentChat,
  AgentChatSnapshot,
  AgentChatStatus,
  AgentDraft,
  AgentInterruptionReason,
  AgentMentionRange,
  AgentMessage,
  AgentPendingAsk,
  AgentResourceReference,
  AgentRunMode,
  Message,
  ToolCall,
} from "../logic/types";
import { getDefaultAgentWindowState } from "../window";

export const AGENT_STATE_KEY = "bb-agent-state-v2";
export const AGENT_WINDOW_KEY = "bb-agent-window";
export const AGENT_WINDOW_LAYOUT_VERSION = 4;

const getWorkspaceStorageKey = (key: string, workspaceName: string) => {
  const workspaceId = getResourceId(workspaceName, workspaceNamePrefix);
  return workspaceId ? `${key}.${workspaceId}` : null;
};

export const getAgentStateKey = (workspaceName: string) =>
  getWorkspaceStorageKey(AGENT_STATE_KEY, workspaceName) ?? AGENT_STATE_KEY;

export const getAgentWindowKey = (workspaceName: string) =>
  getWorkspaceStorageKey(AGENT_WINDOW_KEY, workspaceName) ?? AGENT_WINDOW_KEY;

// This key was used briefly while workspace-scoped persistence was introduced.
// Read it as a migration source so those chats remain recoverable.
const UNASSIGNED_LEGACY_AGENT_STATE_KEY = `${AGENT_STATE_KEY}.legacy`;

interface PersistedAgentState {
  currentChatId: string | null;
  chats: AgentChat[];
  messagesByChatId: Record<string, AgentMessage[]>;
  pendingAskByChatId: Record<string, AgentPendingAsk>;
  draftsByChatId: Record<string, AgentDraft>;
  queuedUserMessage: QueuedUserMessage | null;
}

interface CreateChatOptions {
  page?: AgentChatSnapshot;
  context?: string;
  workspaceName?: string;
  select?: boolean;
  title?: string;
  archived?: boolean;
}

interface AddMessageOptions extends Message {
  chatId?: string;
  metadata?: AgentMessage["metadata"];
}

export interface QueuedUserMessage {
  id: string;
  chatId: string;
  text: string;
  page: AgentChatSnapshot;
  runMode?: AgentRunMode;
  starterId?: LandingStarterId;
  resourceReference?: AgentResourceReference;
  resourceReferences?: AgentResourceReference[];
}

type QueueUserMessageInput = Omit<QueuedUserMessage, "id">;

const DEFAULT_CHAT_STATUS: AgentChatStatus = "idle";
const DRAFT_PERSIST_DELAY_MS = 250;

const normalizeInterruptionReason = (
  value: unknown
): AgentInterruptionReason | undefined =>
  value === "user" || value === "unexpected" ? value : undefined;

const setLastUserInterruptionReason = (
  messages: AgentMessage[],
  reason?: AgentInterruptionReason
) => {
  const lastUserMessage = messages.findLast(
    (message) => message.role === "user"
  );
  if (!lastUserMessage) return;
  if (reason) {
    lastUserMessage.metadata = {
      ...lastUserMessage.metadata,
      interruptionReason: reason,
    };
  } else if (lastUserMessage.metadata?.interruptionReason) {
    const { interruptionReason: _, ...metadata } = lastUserMessage.metadata;
    lastUserMessage.metadata = Object.keys(metadata).length
      ? metadata
      : undefined;
  }
};

// --- Pure helper functions ---

const createChatRecord = (options: CreateChatOptions = {}): AgentChat => {
  const now = Date.now();
  return {
    id: uuidv4(),
    title: options.title ?? "",
    context: options.context,
    workspaceName: options.workspaceName,
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
  return typeof value === "object" && value !== null && !Array.isArray(value);
};

const normalizeDraft = (raw: unknown): AgentDraft | null => {
  if (!isRecord(raw) || typeof raw.text !== "string") return null;
  if (raw.mentions !== undefined) {
    const mentions = normalizeAgentResourceMentions(raw.text, raw.mentions);
    return {
      text: raw.text,
      ...(mentions.length > 0 ? { mentions } : {}),
    };
  }
  const project = isRecord(raw.project) ? raw.project : {};
  const projectName =
    typeof project.name === "string" ? project.name.trim() : "";
  const projectTitle =
    typeof project.title === "string" ? project.title.trim() : "";
  const normalizedProject =
    projectTitle && getNameParentTokens(projectName, [projectNamePrefix]).length
      ? { name: projectName, title: projectTitle }
      : undefined;
  const normalizedResource = normalizeAgentResourceReference(raw.resource);
  const resource =
    normalizedResource &&
    (!normalizedProject ||
      normalizedProject.name === normalizedResource.project)
      ? normalizedResource
      : normalizedProject
        ? {
            kind: "project" as const,
            name: normalizedProject.name,
            project: normalizedProject.name,
            label: normalizedProject.title,
          }
        : undefined;
  if (!resource) return { text: raw.text };

  let text = raw.text;
  let mention: AgentMentionRange | null = null;
  if (isRecord(raw.mention)) {
    const { start, end } = raw.mention;
    if (typeof start === "number" && typeof end === "number") {
      const candidate = { start, end };
      if (isResourceMentionRange(text, candidate, resource.label)) {
        mention = candidate;
      }
    }
    if (!mention) return { text };
  } else if (raw.mention !== undefined) {
    return { text };
  } else {
    const token = formatResourceMention(resource.label);
    let start = text.lastIndexOf(token);
    while (start >= 0) {
      const candidate = { start, end: start + token.length };
      if (isResourceMentionRange(text, candidate, resource.label)) {
        mention = candidate;
        break;
      }
      if (start === 0) break;
      start = text.lastIndexOf(token, start - 1);
    }
    if (!mention) {
      const trigger = getMentionTriggerRange(text, text.length);
      const prefix = trigger
        ? text
        : text && !/\s$/u.test(text)
          ? `${text} @`
          : `${text}@`;
      const inserted = insertResourceMention(
        prefix,
        trigger ?? getMentionTriggerRange(prefix, prefix.length),
        resource.label
      );
      if (!inserted) return { text: raw.text };
      text = inserted.text;
      mention = inserted.mention;
    }
  }
  if (!mention) return { text };

  return {
    text,
    ...(normalizedProject ? { project: normalizedProject } : {}),
    resource,
    mention,
  };
};

const normalizeQueuedUserMessage = (
  raw: unknown,
  chatIds: Set<string>
): QueuedUserMessage | null => {
  if (
    !isRecord(raw) ||
    typeof raw.id !== "string" ||
    !raw.id ||
    typeof raw.chatId !== "string" ||
    !chatIds.has(raw.chatId) ||
    typeof raw.text !== "string" ||
    !raw.text.trim() ||
    !isRecord(raw.page) ||
    typeof raw.page.path !== "string" ||
    typeof raw.page.title !== "string"
  ) {
    return null;
  }
  const resourceReferences = normalizeAgentResourceReferences(
    raw.resourceReferences,
    raw.resourceReference
  );
  return {
    id: raw.id,
    chatId: raw.chatId,
    text: raw.text,
    page: { path: raw.page.path, title: raw.page.title },
    runMode:
      raw.runMode === "default" || raw.runMode === "read-only"
        ? raw.runMode
        : undefined,
    starterId:
      typeof raw.starterId === "string"
        ? (raw.starterId as LandingStarterId)
        : undefined,
    ...(raw.resourceReferences === undefined && resourceReferences[0]
      ? { resourceReference: resourceReferences[0] }
      : {}),
    ...(raw.resourceReferences !== undefined && resourceReferences.length > 0
      ? { resourceReferences }
      : {}),
  };
};

const normalizeMessage = (
  raw: unknown,
  chatId: string,
  fallbackTs: number
): AgentMessage => {
  const message = isRecord(raw) ? raw : {};
  const metadata = isRecord(message.metadata) ? message.metadata : null;
  const resourceReferences = metadata
    ? normalizeAgentResourceReferences(
        metadata.resourceReferences,
        metadata.resourceReference
      )
    : [];
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
    metadata: metadata
      ? {
          ...(metadata as AgentMessage["metadata"]),
          interruptionReason: normalizeInterruptionReason(
            metadata.interruptionReason
          ),
          resourceReference:
            metadata.resourceReferences === undefined
              ? resourceReferences[0]
              : undefined,
          resourceReferences:
            metadata.resourceReferences !== undefined &&
            resourceReferences.length > 0
              ? resourceReferences
              : undefined,
        }
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
    ...(kind === "confirm" &&
    isRecord(pendingAsk.apiPermission) &&
    typeof pendingAsk.apiPermission.actionId === "string" &&
    typeof pendingAsk.apiPermission.operationId === "string"
      ? {
          apiPermission: {
            actionId: pendingAsk.apiPermission.actionId,
            operationId: pendingAsk.apiPermission.operationId,
          },
        }
      : {}),
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
    context:
      typeof chat.context === "string" && chat.context.trim()
        ? chat.context
        : undefined,
    workspaceName:
      typeof chat.workspaceName === "string" && chat.workspaceName.trim()
        ? chat.workspaceName
        : undefined,
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
    interruptionReason: chat.interrupted
      ? (normalizeInterruptionReason(chat.interruptionReason) ?? "unexpected")
      : undefined,
    runId: typeof chat.runId === "string" ? chat.runId : null,
    runMode:
      chat.runMode === "read-only" || chat.runMode === "default"
        ? chat.runMode
        : undefined,
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
  draftsByChatId: {},
  queuedUserMessage: null,
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
  const rawDraftsByChatId = isRecord(raw.draftsByChatId)
    ? raw.draftsByChatId
    : {};

  const chats = rawChats.map((chat) => normalizeChat(chat));
  const messagesByChatId: Record<string, AgentMessage[]> = {};
  const pendingAskByChatId: Record<string, AgentPendingAsk> = {};
  const draftsByChatId: Record<string, AgentDraft> = {};

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
      chat.interruptionReason = "unexpected";
      chat.lastError = null;
      chat.requiresAIConfiguration = false;
    }
    if (chat.interrupted) {
      setLastUserInterruptionReason(
        messages,
        chat.interruptionReason ?? "unexpected"
      );
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
    const draft = normalizeDraft(rawDraftsByChatId[chat.id]);
    if (
      draft &&
      (draft.text || draft.project || draft.resource || draft.mentions)
    ) {
      draftsByChatId[chat.id] = draft;
    }
  }

  const queuedUserMessage = normalizeQueuedUserMessage(
    raw.queuedUserMessage,
    new Set(chats.map((chat) => chat.id))
  );

  const hasCurrentChatId = Object.prototype.hasOwnProperty.call(
    raw,
    "currentChatId"
  );
  const hasCurrentThreadId = Object.prototype.hasOwnProperty.call(
    raw,
    "currentThreadId"
  );
  const persistedCurrentChatId = hasCurrentChatId
    ? raw.currentChatId
    : hasCurrentThreadId
      ? raw.currentThreadId
      : undefined;
  const currentChatId =
    persistedCurrentChatId === null
      ? null
      : typeof persistedCurrentChatId === "string" &&
          chats.some((chat) => chat.id === persistedCurrentChatId)
        ? persistedCurrentChatId
        : (chats[0]?.id ?? null);

  return {
    currentChatId,
    chats,
    messagesByChatId,
    pendingAskByChatId,
    draftsByChatId,
    queuedUserMessage,
  };
};

// --- Zustand store types ---

export interface AgentState {
  // Window state
  visible: boolean;
  size: { width: number; height: number };
  sidebarWidth: number;
  minimized: boolean;

  // Chat state
  workspaceName: string | null;
  chats: AgentChat[];
  messagesByChatId: Record<string, AgentMessage[]>;
  pendingAskByChatId: Record<string, AgentPendingAsk>;
  draftsByChatId: Record<string, AgentDraft>;
  currentChatId: string | null;
  abortControllersByChatId: Record<string, AbortController>;
  queuedUserMessage: QueuedUserMessage | null;
  queuedUserMessageInFlight: boolean;

  // UI actions
  open: () => void;
  toggle: () => void;
  minimize: () => void;
  restore: () => void;
  setSize: (width: number, height: number) => void;
  setSidebarWidth: (width: number) => void;

  // Chat CRUD
  createChat: (options?: CreateChatOptions) => AgentChat;
  queueUserMessage: (message: QueueUserMessageInput) => void;
  takeQueuedUserMessage: (chatId: string) => QueuedUserMessage | null;
  finishQueuedUserMessage: (chatId: string) => void;
  commitQueuedUserMessage: (
    chatId: string
  ) => { queued: QueuedUserMessage; message: AgentMessage } | null;
  getDraft: (chatId?: string | null) => AgentDraft;
  setDraft: (chatId: string, draft: AgentDraft) => void;
  clearDraft: (chatId: string) => void;
  ensureCurrentChat: (
    page?: AgentChatSnapshot,
    context?: string,
    workspaceName?: string
  ) => AgentChat;
  selectChatContext: (
    context: string,
    page?: AgentChatSnapshot,
    workspaceName?: string
  ) => AgentChat | null;
  selectOrCreateChatForContext: (
    context: string,
    page?: AgentChatSnapshot,
    workspaceName?: string
  ) => AgentChat | null;
  setCurrentChat: (chatId: string) => void;
  clearCurrentChat: () => void;
  updateChatPage: (chatId: string, page: AgentChatSnapshot) => AgentChat | null;
  renameChat: (chatId: string, title: string) => AgentChat | null;
  archiveChat: (chatId: string) => AgentChat | null;
  archiveAllActiveChats: () => number;
  unarchiveChat: (chatId: string) => AgentChat | null;
  unarchiveAllArchivedChats: () => number;
  deleteChat: (chatId: string) => boolean;
  deleteAllArchivedChats: () => number;

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
      interruptionReason?: AgentInterruptionReason;
      lastError?: string | null;
      requiresAIConfiguration?: boolean;
      page?: AgentChatSnapshot;
    }
  ) => AgentChat | null;
  startChatRun: (
    chatId: string,
    page?: AgentChatSnapshot,
    options?: { runId?: string; runMode?: AgentRunMode }
  ) => void;
  finishChatRun: (
    chatId: string,
    options?: {
      status?: Extract<AgentChatStatus, "idle" | "error">;
      lastError?: string | null;
      requiresAIConfiguration?: boolean;
    }
  ) => void;
  interruptChatRun: (
    chatId: string,
    page?: AgentChatSnapshot,
    reason?: AgentInterruptionReason
  ) => void;

  // Pending ask
  setPendingAsk: (
    chatId: string,
    pendingAsk: AgentPendingAsk
  ) => AgentPendingAsk;
  clearPendingAsk: (chatId?: string | null) => void;
  awaitUser: (chatId: string, pendingAsk: AgentPendingAsk) => AgentPendingAsk;
  cancelPendingAsk: (chatId?: string | null) => boolean;
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
  cancel: (chatId?: string | null, reason?: AgentInterruptionReason) => void;

  // Error
  clearError: (chatId?: string | null) => void;

  // Persistence
  setWorkspaceName: (workspaceName?: string) => boolean;
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

const getInitialWindowState = () => {
  if (typeof window === "undefined") {
    return {
      size: { width: 480, height: 540 },
      sidebarWidth: 200,
    };
  }

  return getDefaultAgentWindowState(window.innerWidth, window.innerHeight);
};

const ensureCurrentChat = (state: PersistedAgentState): PersistedAgentState => {
  if (
    state.currentChatId === null
      ? state.chats.length > 0
      : state.chats.some((chat) => chat.id === state.currentChatId)
  ) {
    return state;
  }

  const defaultChat = createChatRecord();
  return {
    ...state,
    chats: [...state.chats, defaultChat],
    messagesByChatId: {
      ...state.messagesByChatId,
      [defaultChat.id]: [],
    },
    currentChatId: defaultChat.id,
  };
};

const readPersistedState = (
  storage: Storage | null,
  key: string
): PersistedAgentState | null => {
  const saved = storage?.getItem(key);
  if (!saved) return null;
  try {
    return normalizePersistedState(JSON.parse(saved));
  } catch {
    storage?.removeItem(key);
    return null;
  }
};

const getPersistedState = (storage: Storage | null, key: string) =>
  ensureCurrentChat(
    readPersistedState(storage, key) ?? createEmptyPersistedState()
  );

const getPersistedStateForChats = (
  state: PersistedAgentState,
  chats: AgentChat[]
): PersistedAgentState => {
  const chatIds = new Set(chats.map((chat) => chat.id));
  return {
    currentChatId: chatIds.has(state.currentChatId ?? "")
      ? state.currentChatId
      : (chats[0]?.id ?? null),
    chats,
    messagesByChatId: Object.fromEntries(
      chats.map((chat) => [chat.id, state.messagesByChatId[chat.id] ?? []])
    ),
    pendingAskByChatId: Object.fromEntries(
      chats.flatMap((chat) => {
        const pendingAsk = state.pendingAskByChatId[chat.id];
        return pendingAsk ? [[chat.id, pendingAsk]] : [];
      })
    ),
    draftsByChatId: Object.fromEntries(
      chats.flatMap((chat) => {
        const draft = state.draftsByChatId[chat.id];
        return draft ? [[chat.id, draft]] : [];
      })
    ),
    queuedUserMessage: chatIds.has(state.queuedUserMessage?.chatId ?? "")
      ? state.queuedUserMessage
      : null,
  };
};

const mergePersistedStates = (
  existing: PersistedAgentState | null,
  incoming: PersistedAgentState
) => {
  if (!existing) return incoming;
  const chatsById = new Map(existing.chats.map((chat) => [chat.id, chat]));
  for (const chat of incoming.chats) {
    const current = chatsById.get(chat.id);
    if (!current || current.updatedTs < chat.updatedTs) {
      chatsById.set(chat.id, chat);
    }
  }
  const chats = [...chatsById.values()];
  const messagesByChatId = {
    ...existing.messagesByChatId,
    ...incoming.messagesByChatId,
  };
  const pendingAskByChatId = {
    ...existing.pendingAskByChatId,
    ...incoming.pendingAskByChatId,
  };
  const draftsByChatId = {
    ...existing.draftsByChatId,
    ...incoming.draftsByChatId,
  };
  const queuedUserMessage =
    existing.queuedUserMessage ?? incoming.queuedUserMessage;
  return {
    currentChatId:
      existing.currentChatId && chatsById.has(existing.currentChatId)
        ? existing.currentChatId
        : incoming.currentChatId,
    chats,
    messagesByChatId,
    pendingAskByChatId,
    draftsByChatId,
    queuedUserMessage,
  };
};

const migrateLegacyState = (
  storage: Storage | null,
  fallbackWorkspaceName: string
) => {
  if (!storage) return;
  const sourceKeys = [AGENT_STATE_KEY, UNASSIGNED_LEGACY_AGENT_STATE_KEY];
  const legacyState = sourceKeys
    .map((key) => readPersistedState(storage, key))
    .reduce<PersistedAgentState | null>(
      (state, next) => (next ? mergePersistedStates(state, next) : state),
      null
    );
  if (!legacyState) return;

  const chatsByWorkspace = new Map<string, AgentChat[]>();
  for (const chat of legacyState.chats) {
    const key = chat.workspaceName
      ? getWorkspaceStorageKey(AGENT_STATE_KEY, chat.workspaceName)
      : null;
    if (!key) {
      const fallbackKey = getWorkspaceStorageKey(
        AGENT_STATE_KEY,
        fallbackWorkspaceName
      );
      if (!fallbackKey) return;
      const chats = chatsByWorkspace.get(fallbackKey) ?? [];
      // Legacy state has no trustworthy project-to-workspace association.
      chats.push({
        ...chat,
        workspaceName: fallbackWorkspaceName,
        context: fallbackWorkspaceName,
      });
      chatsByWorkspace.set(fallbackKey, chats);
    } else {
      const chats = chatsByWorkspace.get(key) ?? [];
      chats.push(chat);
      chatsByWorkspace.set(key, chats);
    }
  }

  try {
    for (const [key, chats] of chatsByWorkspace) {
      const incoming = getPersistedStateForChats(legacyState, chats);
      const existing = readPersistedState(storage, key);
      storage.setItem(
        key,
        JSON.stringify(mergePersistedStates(existing, incoming))
      );
    }
    for (const key of sourceKeys) storage.removeItem(key);
  } catch {
    // Keep the source cache until every target cache has been written.
  }
};

const loadInitialState = (): {
  chats: AgentChat[];
  messagesByChatId: Record<string, AgentMessage[]>;
  pendingAskByChatId: Record<string, AgentPendingAsk>;
  draftsByChatId: Record<string, AgentDraft>;
  queuedUserMessage: QueuedUserMessage | null;
  currentChatId: string | null;
} => {
  return ensureCurrentChat(createEmptyPersistedState());
};

export const createAgentStore = () => {
  const initial = loadInitialState();
  const initialWindowState = getInitialWindowState();
  const lastSerializedByWorkspace = new Map<string, string>();
  let draftPersistTimer: ReturnType<typeof setTimeout> | null = null;
  const clearDraftPersistTimer = () => {
    if (draftPersistTimer !== null) {
      clearTimeout(draftPersistTimer);
      draftPersistTimer = null;
    }
  };

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
        size: initialWindowState.size,
        sidebarWidth: initialWindowState.sidebarWidth,
        minimized: false,

        // Chat state (from persisted/initial)
        workspaceName: null,
        chats: initial.chats,
        messagesByChatId: initial.messagesByChatId,
        pendingAskByChatId: initial.pendingAskByChatId,
        draftsByChatId: initial.draftsByChatId,
        currentChatId: initial.currentChatId,
        abortControllersByChatId: {},
        queuedUserMessage: initial.queuedUserMessage,
        queuedUserMessageInFlight: false,

        // UI actions
        open: () =>
          set((state) => {
            state.visible = true;
            state.minimized = false;
          }),
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

        getDraft: (chatId) => {
          const id = chatId ?? get().currentChatId;
          return (id ? get().draftsByChatId[id] : undefined) ?? { text: "" };
        },

        setDraft: (chatId, draft) => {
          if (!_getChat(chatId)) return;
          const normalized = normalizeDraft(draft);
          if (!normalized) return;
          set((state) => {
            if (
              normalized.text ||
              normalized.project ||
              normalized.resource ||
              normalized.mentions
            ) {
              state.draftsByChatId[chatId] = normalized;
            } else {
              delete state.draftsByChatId[chatId];
            }
          });
        },

        clearDraft: (chatId) =>
          set((state) => {
            delete state.draftsByChatId[chatId];
          }),

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

        queueUserMessage: (message) => {
          const { resourceReference, resourceReferences, ...rest } = message;
          const normalizedReferences = normalizeAgentResourceReferences(
            resourceReferences,
            resourceReference
          );
          set((state) => {
            state.queuedUserMessage = {
              ...rest,
              id: uuidv4(),
              ...(resourceReferences === undefined && normalizedReferences[0]
                ? { resourceReference: normalizedReferences[0] }
                : {}),
              ...(resourceReferences !== undefined &&
              normalizedReferences.length > 0
                ? { resourceReferences: normalizedReferences }
                : {}),
            };
            state.queuedUserMessageInFlight = false;
          });
        },

        takeQueuedUserMessage: (chatId) => {
          const message = get().queuedUserMessage;
          if (message?.chatId !== chatId || get().queuedUserMessageInFlight) {
            return null;
          }
          set((state) => {
            state.queuedUserMessageInFlight = true;
          });
          return message;
        },

        finishQueuedUserMessage: (chatId) =>
          set((state) => {
            if (state.queuedUserMessage?.chatId !== chatId) return;
            state.queuedUserMessage = null;
            state.queuedUserMessageInFlight = false;
          }),

        commitQueuedUserMessage: (chatId) => {
          const queued = get().queuedUserMessage;
          if (!queued || queued.chatId !== chatId || !_getChat(chatId)) {
            return null;
          }

          let message: AgentMessage | null = null;
          set((state) => {
            const queued = state.queuedUserMessage;
            if (!queued || queued.chatId !== chatId) return;
            const chatMessages = state.messagesByChatId[chatId] ?? [];
            state.messagesByChatId[chatId] = chatMessages;
            const existing = chatMessages.find(
              (candidate) => candidate.id === queued.id
            );
            if (existing) {
              if (existing.role !== "user") return;
              message = { ...existing } as AgentMessage;
            } else {
              const committed: AgentMessage = {
                id: queued.id,
                chatId,
                createdTs: Date.now(),
                role: "user",
                content: queued.text,
                metadata: queued.resourceReferences
                  ? { resourceReferences: queued.resourceReferences }
                  : queued.resourceReference
                    ? { resourceReference: queued.resourceReference }
                    : undefined,
              };
              chatMessages.push(committed);
              message = committed;
              const chat = state.chats.find(
                (candidate) => candidate.id === chatId
              );
              if (chat) {
                if (!chat.title)
                  chat.title = getChatTitleFromMessage(queued.text);
                chat.lastError = null;
                chat.requiresAIConfiguration = false;
                chat.interrupted = false;
                chat.interruptionReason = undefined;
                chat.updatedTs = committed.createdTs;
              }
            }
            state.queuedUserMessageInFlight = true;
          });
          return message ? { queued, message } : null;
        },

        ensureCurrentChat: (page, context, workspaceName) => {
          const existing = _getChat(get().currentChatId);
          if (existing) {
            const canBindContext =
              Boolean(context) &&
              !existing.context &&
              get().getMessages(existing.id).length === 0;
            if ((page && !existing.page) || canBindContext) {
              set((state) => {
                const chat = state.chats.find((c) => c.id === existing.id);
                if (chat) {
                  if (page && !chat.page) chat.page = page;
                  if (canBindContext && context) {
                    chat.context = context;
                    chat.workspaceName = workspaceName;
                  }
                  _touchChatInDraft(state.chats, chat.id);
                }
              });
            }
            return _getChat(existing.id)!;
          }
          return get().createChat({ page, context, workspaceName });
        },

        selectChatContext: (context, page, workspaceName) => {
          if (!context) return null;
          if (
            workspaceName &&
            get().workspaceName !== workspaceName &&
            !get().setWorkspaceName(workspaceName)
          ) {
            return null;
          }
          if (_hasRunningChat()) return null;
          const current = _getChat(get().currentChatId);
          if (!current)
            return get().createChat({ page, context, workspaceName });
          if (
            current.context === context &&
            current.workspaceName === workspaceName
          ) {
            return current;
          }
          if (get().getPendingAsk(current.id)) return null;
          set((state) => {
            const chat = state.chats.find((c) => c.id === current.id);
            if (chat) {
              chat.context = context;
              chat.workspaceName = workspaceName;
              _touchChatInDraft(state.chats, chat.id);
            }
          });
          return _getChat(current.id);
        },

        selectOrCreateChatForContext: (context, page, workspaceName) => {
          if (!context || _hasRunningChat() || get().queuedUserMessage) {
            return null;
          }
          if (
            workspaceName &&
            get().workspaceName !== workspaceName &&
            !get().setWorkspaceName(workspaceName)
          ) {
            return null;
          }

          const isReusable = (chat: AgentChat) =>
            !chat.archived &&
            chat.status === "idle" &&
            !get().getPendingAsk(chat.id);
          const current = _getChat(get().currentChatId);
          const matchingChat =
            current?.context === context &&
            current.workspaceName === workspaceName &&
            isReusable(current)
              ? current
              : [...get().chats]
                  .filter(
                    (chat) =>
                      chat.context === context &&
                      chat.workspaceName === workspaceName &&
                      isReusable(chat)
                  )
                  .sort(
                    (a, b) =>
                      b.updatedTs - a.updatedTs || b.createdTs - a.createdTs
                  )[0];

          if (matchingChat) {
            set((state) => {
              const chat = state.chats.find((c) => c.id === matchingChat.id);
              if (!chat) return;
              if (page) chat.page = page;
              state.currentChatId = chat.id;
              _touchChatInDraft(state.chats, chat.id);
            });
            return _getChat(matchingChat.id);
          }

          if (
            current &&
            !current.context &&
            isReusable(current) &&
            get().getMessages(current.id).length === 0
          ) {
            set((state) => {
              const chat = state.chats.find((c) => c.id === current.id);
              if (!chat) return;
              chat.context = context;
              chat.workspaceName = workspaceName;
              if (page) chat.page = page;
              _touchChatInDraft(state.chats, chat.id);
            });
            return _getChat(current.id);
          }

          return get().createChat({ context, page, workspaceName });
        },

        setCurrentChat: (chatId) => {
          if (get().canSelectChat(chatId)) {
            set((state) => {
              state.currentChatId = chatId;
            });
          }
        },

        clearCurrentChat: () => {
          set((state) => {
            state.currentChatId = null;
          });
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

        archiveAllActiveChats: () => {
          let archivedCount = 0;
          set((state) => {
            const activeChats = state.chats.filter((chat) => !chat.archived);
            archivedCount = activeChats.length;
            if (archivedCount === 0) return;

            for (const chat of activeChats) {
              chat.archived = true;
              _touchChatInDraft(state.chats, chat.id);
            }

            if (
              state.currentChatId &&
              activeChats.some((chat) => chat.id === state.currentChatId)
            ) {
              state.currentChatId = null;
            }
          });
          return archivedCount;
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

        unarchiveAllArchivedChats: () => {
          let unarchivedCount = 0;
          set((state) => {
            const archivedChats = state.chats.filter((chat) => chat.archived);
            unarchivedCount = archivedChats.length;
            if (unarchivedCount === 0) return;

            for (const chat of archivedChats) {
              chat.archived = false;
              _touchChatInDraft(state.chats, chat.id);
            }

            if (
              state.currentChatId &&
              archivedChats.some((chat) => chat.id === state.currentChatId)
            ) {
              state.currentChatId = null;
            }
          });
          return unarchivedCount;
        },

        deleteChat: (chatId) => {
          if (!_getChat(chatId)) return false;
          get().cancel(chatId);
          set((state) => {
            state.chats = state.chats.filter((c) => c.id !== chatId);
            delete state.messagesByChatId[chatId];
            delete state.pendingAskByChatId[chatId];
            delete state.draftsByChatId[chatId];
            delete state.abortControllersByChatId[chatId];
            if (state.queuedUserMessage?.chatId === chatId) {
              state.queuedUserMessage = null;
              state.queuedUserMessageInFlight = false;
            }
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

        deleteAllArchivedChats: () => {
          const archivedChatIds = get()
            .chats.filter((chat) => chat.archived)
            .map((chat) => chat.id);
          for (const chatId of archivedChatIds) {
            get().cancel(chatId);
          }

          let deletedCount = 0;
          set((state) => {
            const archivedChats = state.chats.filter((chat) => chat.archived);
            deletedCount = archivedChats.length;
            if (deletedCount === 0) return;

            const deletedChatIds = new Set(
              archivedChats.map((chat) => chat.id)
            );
            for (const chatId of deletedChatIds) {
              delete state.messagesByChatId[chatId];
              delete state.pendingAskByChatId[chatId];
              delete state.draftsByChatId[chatId];
              delete state.abortControllersByChatId[chatId];
            }

            if (
              state.queuedUserMessage &&
              deletedChatIds.has(state.queuedUserMessage.chatId)
            ) {
              state.queuedUserMessage = null;
              state.queuedUserMessageInFlight = false;
            }

            state.chats = state.chats.filter(
              (chat) => !deletedChatIds.has(chat.id)
            );
            if (
              state.currentChatId &&
              deletedChatIds.has(state.currentChatId)
            ) {
              state.currentChatId = null;
            }
          });
          return deletedCount;
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
              if (message.role === "user") {
                chat.interrupted = false;
                chat.interruptionReason = undefined;
              }
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
              chat.interruptionReason = chat.interrupted
                ? (options.interruptionReason ?? "unexpected")
                : undefined;
              chat.lastError = options.lastError ?? null;
              chat.requiresAIConfiguration =
                options.requiresAIConfiguration ?? false;
              if (options.page) {
                chat.page = options.page;
              }
              if (status !== "awaiting_user") {
                delete state.pendingAskByChatId[chatId];
              }
              if (
                status !== "running" &&
                status !== "awaiting_user" &&
                state.queuedUserMessage?.chatId === chatId &&
                state.messagesByChatId[chatId]?.some(
                  (message) =>
                    message.id === state.queuedUserMessage?.id &&
                    message.role === "user"
                )
              ) {
                state.queuedUserMessage = null;
                state.queuedUserMessageInFlight = false;
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
              chat.runMode = options.runMode;
              setLastUserInterruptionReason(
                state.messagesByChatId[chatId] ?? []
              );
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
              chat.runMode = undefined;
            }
          });
        },

        interruptChatRun: (chatId, page, reason) => {
          const interruptionReason =
            reason === "user" || _getChat(chatId)?.interruptionReason === "user"
              ? "user"
              : "unexpected";
          get().setChatStatus(chatId, DEFAULT_CHAT_STATUS, {
            interrupted: true,
            interruptionReason,
            page,
          });
          set((state) => {
            setLastUserInterruptionReason(
              state.messagesByChatId[chatId] ?? [],
              interruptionReason
            );
          });
        },

        // Pending ask
        setPendingAsk: (chatId, pendingAsk) => {
          set((state) => {
            state.pendingAskByChatId[chatId] = pendingAsk;
            if (
              state.chats.find((chat) => chat.id === chatId)?.status ===
                "awaiting_user" &&
              state.queuedUserMessage?.chatId === chatId &&
              state.messagesByChatId[chatId]?.some(
                (message) =>
                  message.id === state.queuedUserMessage?.id &&
                  message.role === "user"
              )
            ) {
              state.queuedUserMessage = null;
              state.queuedUserMessageInFlight = false;
            }
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

        cancelPendingAsk: (chatId) => {
          const id = chatId ?? get().currentChatId;
          const chat = _getChat(id);
          const pendingAsk = id ? get().pendingAskByChatId[id] : undefined;
          if (
            !id ||
            chat?.status !== "awaiting_user" ||
            !pendingAsk ||
            pendingAsk.apiPermission
          ) {
            return false;
          }
          const toolMessage: AgentMessage = {
            id: uuidv4(),
            chatId: id,
            createdTs: Date.now(),
            role: "tool",
            toolCallId: pendingAsk.toolCallId,
            content: JSON.stringify({ kind: "cancelled", reason: "user" }),
            metadata: chat.runId ? { runId: chat.runId } : undefined,
          };
          set((state) => {
            const currentChat = state.chats.find(
              (candidate) => candidate.id === id
            );
            if (!currentChat) return;
            delete state.pendingAskByChatId[id];
            state.messagesByChatId[id] ??= [];
            state.messagesByChatId[id].push(toolMessage);
            currentChat.status = DEFAULT_CHAT_STATUS;
            currentChat.runId = null;
            currentChat.runMode = undefined;
            currentChat.interrupted = false;
            currentChat.interruptionReason = undefined;
            currentChat.lastError = null;
            setLastUserInterruptionReason(state.messagesByChatId[id] ?? []);
            _touchChatInDraft(state.chats, id);
          });
          return true;
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
        cancel: (chatId, reason) => {
          const id = chatId ?? get().currentChatId;
          if (!id) return;
          get().getAbortController(id)?.abort();
          get().setAbortController(id, null);
          if (get().isChatRunning(id)) {
            get().interruptChatRun(id, undefined, reason);
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
              c.interruptionReason = undefined;
              c.runId = null;
              if (c.status === "error") {
                c.status = DEFAULT_CHAT_STATUS;
              }
              _touchChatInDraft(state.chats, c.id);
            }
          });
        },

        // Persistence
        setWorkspaceName: (workspaceName) => {
          if (!workspaceName || get().workspaceName === workspaceName) {
            return Boolean(workspaceName);
          }
          if (_hasRunningChat() || get().queuedUserMessage) {
            return false;
          }
          const stateKey = getWorkspaceStorageKey(
            AGENT_STATE_KEY,
            workspaceName
          );
          if (!stateKey) return false;

          get().saveState();
          const storage = getSafeStorage();
          migrateLegacyState(storage, workspaceName);
          const persistedState = getPersistedState(storage, stateKey);
          set((draft) => {
            draft.workspaceName = workspaceName;
            draft.chats = persistedState.chats;
            draft.messagesByChatId = persistedState.messagesByChatId;
            draft.pendingAskByChatId = persistedState.pendingAskByChatId;
            draft.draftsByChatId = persistedState.draftsByChatId;
            draft.currentChatId = persistedState.currentChatId;
            draft.abortControllersByChatId = {};
            draft.queuedUserMessage = persistedState.queuedUserMessage;
            draft.queuedUserMessageInFlight = false;
          });
          get().loadWindowState();
          return true;
        },

        saveWindowState: () => {
          const storage = getSafeStorage();
          const workspaceName = get().workspaceName;
          const key = workspaceName
            ? getWorkspaceStorageKey(AGENT_WINDOW_KEY, workspaceName)
            : null;
          if (!storage || !key) return;
          const state = get();
          storage.setItem(
            key,
            JSON.stringify({
              layoutVersion: AGENT_WINDOW_LAYOUT_VERSION,
              size: state.size,
              sidebarWidth: state.sidebarWidth,
            })
          );
        },

        loadWindowState: () => {
          const storage = getSafeStorage();
          const workspaceName = get().workspaceName;
          const key = workspaceName
            ? getWorkspaceStorageKey(AGENT_WINDOW_KEY, workspaceName)
            : null;
          if (!key) return;
          const saved =
            storage?.getItem(key) ?? storage?.getItem(AGENT_WINDOW_KEY);
          if (!saved) {
            const defaultWindowState = getInitialWindowState();
            set((draft) => {
              draft.size = defaultWindowState.size;
              draft.sidebarWidth = defaultWindowState.sidebarWidth;
            });
            return;
          }
          try {
            const state = JSON.parse(saved) as {
              layoutVersion?: number;
              position?: unknown;
              size?: { width?: number; height?: number };
              sidebarWidth?: number;
            };
            const isCurrentLayout =
              state.layoutVersion === AGENT_WINDOW_LAYOUT_VERSION;
            const hasLegacyPosition = state.position !== undefined;
            const defaultWindowState = getInitialWindowState();
            set((draft) => {
              if (
                isCurrentLayout &&
                typeof state.size?.width === "number" &&
                typeof state.size?.height === "number"
              ) {
                draft.size = {
                  width: state.size.width,
                  height: state.size.height,
                };
              } else if (!isCurrentLayout) {
                draft.size = defaultWindowState.size;
              }
              if (typeof state.sidebarWidth === "number") {
                draft.sidebarWidth = state.sidebarWidth;
              }
            });
            if (
              !isCurrentLayout ||
              !storage?.getItem(key) ||
              hasLegacyPosition
            ) {
              get().saveWindowState();
            }
          } catch {
            storage?.removeItem(key);
          }
        },

        saveState: () => {
          clearDraftPersistTimer();
          const storage = getSafeStorage();
          const workspaceName = get().workspaceName;
          const key = workspaceName
            ? getWorkspaceStorageKey(AGENT_STATE_KEY, workspaceName)
            : null;
          if (!storage || !key) return;
          const state = get();
          const persistedState: PersistedAgentState = {
            currentChatId: state.currentChatId,
            chats: state.chats,
            messagesByChatId: state.messagesByChatId,
            pendingAskByChatId: state.pendingAskByChatId,
            draftsByChatId: state.draftsByChatId,
            queuedUserMessage: state.queuedUserMessage,
          };
          const serialized = JSON.stringify(persistedState);
          lastSerializedByWorkspace.set(key, serialized);
          storage.setItem(key, serialized);
        },

        loadState: () => {
          const storage = getSafeStorage();
          const workspaceName = get().workspaceName;
          const key = workspaceName
            ? getWorkspaceStorageKey(AGENT_STATE_KEY, workspaceName)
            : null;
          if (!key) return;
          const state = getPersistedState(storage, key);
          set((draft) => {
            draft.chats = state.chats;
            draft.messagesByChatId = state.messagesByChatId;
            draft.pendingAskByChatId = state.pendingAskByChatId;
            draft.draftsByChatId = state.draftsByChatId;
            draft.currentChatId = state.currentChatId;
            draft.queuedUserMessage = state.queuedUserMessage;
            draft.queuedUserMessageInFlight = false;
          });
        },
      };
    })
  );

  const persistState = (state: AgentState) => {
    const storage = getSafeStorage();
    const workspaceName = state.workspaceName;
    const key = workspaceName
      ? getWorkspaceStorageKey(AGENT_STATE_KEY, workspaceName)
      : null;
    if (!storage || !key) {
      return;
    }
    const persistedState: PersistedAgentState = {
      currentChatId: state.currentChatId,
      chats: state.chats,
      messagesByChatId: state.messagesByChatId,
      pendingAskByChatId: state.pendingAskByChatId,
      draftsByChatId: state.draftsByChatId,
      queuedUserMessage: state.queuedUserMessage,
    };
    const serialized = JSON.stringify(persistedState);
    if (serialized !== lastSerializedByWorkspace.get(key)) {
      lastSerializedByWorkspace.set(key, serialized);
      storage.setItem(key, serialized);
    }
  };

  store.subscribe((state, previousState) => {
    const nonDraftStateChanged =
      state.workspaceName !== previousState.workspaceName ||
      state.currentChatId !== previousState.currentChatId ||
      state.chats !== previousState.chats ||
      state.messagesByChatId !== previousState.messagesByChatId ||
      state.pendingAskByChatId !== previousState.pendingAskByChatId ||
      state.queuedUserMessage !== previousState.queuedUserMessage;
    const draftChanged = state.draftsByChatId !== previousState.draftsByChatId;
    if (!nonDraftStateChanged && !draftChanged) return;

    clearDraftPersistTimer();
    if (!nonDraftStateChanged && draftChanged) {
      draftPersistTimer = setTimeout(() => {
        draftPersistTimer = null;
        persistState(store.getState());
      }, DRAFT_PERSIST_DELAY_MS);
    } else {
      persistState(state);
    }
  });

  return store;
};

// Singleton for production use.
// The return value of create() IS a React hook — callable as
// useAgentStore(selector) to subscribe React components to state slices.
// Also callable outside React via useAgentStore.getState().
export const useAgentStore = createAgentStore();

if (typeof window !== "undefined") {
  window.addEventListener("pagehide", () =>
    useAgentStore.getState().saveState()
  );
}
