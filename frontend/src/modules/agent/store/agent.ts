import { v4 as uuidv4 } from "uuid";
import { create } from "zustand";
import { immer } from "zustand/middleware/immer";
import { getResourceId, workspaceNamePrefix } from "@/lib/resourceName";
import {
  getNameParentTokens,
  projectNamePrefix,
} from "@/stores/modules/v1/common";
import { normalizeAgentDiagnosticCapability } from "../logic/diagnosticCapability";
import type {
  AgentExpressionDraftMode,
  AgentExpressionDraftTarget,
} from "../logic/expressionDraft";
import { normalizeAgentHandoff } from "../logic/handoff";
import type { LandingStarterId } from "../logic/landingStarters";
import {
  formatResourceMention,
  getCurrentTurnResourceReferences,
  getMentionTriggerRange,
  insertResourceMention,
  isResourceMentionRange,
  normalizeAgentResourceMentions,
  normalizeAgentResourceReference,
  normalizeAgentResourceReferences,
} from "../logic/resourceMention";
import { normalizeAgentResponseActions } from "../logic/responseActions";
import {
  hasAgentRunLease,
  supportsAgentRunRecovery,
} from "../logic/runOwnership";
import {
  createRunBudget,
  increaseRunBudgetLimit,
  mergeModelAttempts,
  normalizeModelAttempt,
  normalizeRunBindings,
  normalizeRunBudget,
  normalizeRunCheckpoint,
  projectChatUsage,
  projectRunBudget,
  reserveRunAttempt,
} from "../logic/tokenBudget";
import { normalizeModelCallUsage } from "../logic/tokenUsage";
import type {
  AgentAskUserResponse,
  AgentAttemptReservation,
  AgentChat,
  AgentChatSnapshot,
  AgentChatStatus,
  AgentDiagnosticCapability,
  AgentDraft,
  AgentInterruptionReason,
  AgentMentionRange,
  AgentMessage,
  AgentMessageMetadata,
  AgentModelAttempt,
  AgentModelCallUsage,
  AgentPendingAsk,
  AgentResourceReference,
  AgentResponseAction,
  AgentRunBindings,
  AgentRunCheckpoint,
  AgentRunMode,
  Message,
  ToolCall,
} from "../logic/types";
import { getDefaultAgentWindowState } from "../window";
import {
  type AgentPreferences,
  DEFAULT_AGENT_PREFERENCES,
  getAgentPreferencesKey,
  normalizeAgentPreferences,
} from "./preferences";
import { createAgentRunPersistence } from "./runPersistence";

export const AGENT_STATE_KEY = "bb-agent-state-v2";
export const AGENT_RUN_STATE_KEY = "bb-agent-state-v3";
export const AGENT_WINDOW_KEY = "bb-agent-window";
export const AGENT_WINDOW_LAYOUT_VERSION = 4;
const LEGACY_AGENT_STATE_KEY = `${AGENT_STATE_KEY}.legacy`;
const LEGACY_AGENT_OWNER_KEY = `${AGENT_STATE_KEY}-migration-owner`;

const getWorkspaceStorageKey = (key: string, workspaceName: string) => {
  const workspaceId = getResourceId(workspaceName, workspaceNamePrefix);
  return workspaceId ? `${key}.${workspaceId}` : null;
};

export const getAgentStateKey = (workspaceName: string, userEmail: string) => {
  const workspaceId = getResourceId(workspaceName, workspaceNamePrefix);
  return workspaceId && userEmail
    ? `${supportsAgentRunRecovery() ? AGENT_RUN_STATE_KEY : AGENT_STATE_KEY}.${workspaceId.length}:${workspaceId}.${userEmail}`
    : null;
};

export const getAgentWindowKey = (workspaceName: string) =>
  getWorkspaceStorageKey(AGENT_WINDOW_KEY, workspaceName) ?? AGENT_WINDOW_KEY;

interface PersistedAgentState {
  currentChatId: string | null;
  chats: AgentChat[];
  messagesByChatId: Record<string, AgentMessage[]>;
  pendingAskByChatId: Record<string, AgentPendingAsk>;
  draftsByChatId: Record<string, AgentDraft>;
  queuedUserMessagesByChatId: Record<string, QueuedUserMessage>;
  activeChatIds?: string[];
}

interface CreateChatOptions {
  page?: AgentChatSnapshot;
  context?: string;
  workspaceName?: string;
  select?: boolean;
  title?: string;
  archived?: boolean;
  capability?: AgentDiagnosticCapability;
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

interface RegisteredExpressionDraftTarget {
  apply: AgentExpressionDraftTarget;
  allowRaw: boolean;
  token: symbol;
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
  const capability = normalizeAgentDiagnosticCapability(options.capability);
  if (options.capability !== undefined && !capability) {
    throw new Error("Invalid Agent diagnostic capability");
  }
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
    ...(capability ? { capability } : {}),
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

type DraftHandoffKind =
  | "draft-masking-rule"
  | "draft-masking-exemption"
  | "draft-semantic-type";

const normalizeDraftHandoff = (
  receipt: unknown,
  kind: DraftHandoffKind,
  role: unknown,
  actions: AgentResponseAction[]
): AgentMessageMetadata["maskingExemptionDraftHandoff"] =>
  role === "assistant" &&
  isRecord(receipt) &&
  typeof receipt.draftId === "string" &&
  (receipt.outcome === "opened" || receipt.outcome === "applied") &&
  (kind !== "draft-semantic-type" || receipt.outcome === "opened") &&
  actions.some(
    (action) => action.kind === kind && action.draftId === receipt.draftId
  )
    ? {
        draftId: receipt.draftId,
        outcome: receipt.outcome,
        ...(kind !== "draft-masking-rule" && receipt.created === true
          ? { created: true as const }
          : {}),
      }
    : undefined;

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
  const actions = normalizeAgentResponseActions(metadata?.actions);
  const semanticTypeReceipt = normalizeDraftHandoff(
    metadata?.semanticTypeDraftHandoff,
    "draft-semantic-type",
    message.role,
    actions
  );
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
          handoff: normalizeAgentHandoff(metadata.handoff),
          actions: actions.length > 0 ? actions : undefined,
          maskingRuleDraftHandoff: normalizeDraftHandoff(
            metadata.maskingRuleDraftHandoff,
            "draft-masking-rule",
            message.role,
            actions
          ),
          maskingExemptionDraftHandoff: normalizeDraftHandoff(
            metadata.maskingExemptionDraftHandoff,
            "draft-masking-exemption",
            message.role,
            actions
          ),
          semanticTypeDraftHandoff:
            semanticTypeReceipt?.outcome === "opened"
              ? { ...semanticTypeReceipt, outcome: "opened" }
              : undefined,
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

const normalizePendingAsk = (
  raw: unknown,
  context?: string
): AgentPendingAsk | null => {
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
  const defaultResource = normalizeAgentResourceReference(
    pendingAsk.defaultResource
  );
  const hasBoundDefault =
    kind === "input" &&
    defaultResource?.kind === "database" &&
    defaultResource.name === pendingAsk.defaultValue &&
    (!context?.startsWith(projectNamePrefix) ||
      defaultResource.project === context);

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
    ...(hasBoundDefault ? { defaultResource } : {}),
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
    chat.status === "paused" ||
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
    hasUnreportedUsage: chat.hasUnreportedUsage === true,
    legacyUsage:
      isRecord(chat.legacyUsage) &&
      typeof chat.legacyUsage.totalTokensUsed === "number" &&
      Number.isSafeInteger(chat.legacyUsage.totalTokensUsed) &&
      chat.legacyUsage.totalTokensUsed >= 0
        ? {
            totalTokensUsed: chat.legacyUsage.totalTokensUsed,
            hasUnreportedUsage: chat.legacyUsage.hasUnreportedUsage === true,
          }
        : undefined,
    modelAttempts: Array.isArray(chat.modelAttempts)
      ? mergeModelAttempts(chat.modelAttempts as AgentModelAttempt[])
      : undefined,
    runBudget: normalizeRunBudget(chat.runBudget),
    runCheckpoint: normalizeRunCheckpoint(chat.runCheckpoint),
    lastRunUsage:
      isRecord(chat.lastRunUsage) &&
      typeof chat.lastRunUsage.runId === "string" &&
      Array.isArray(chat.lastRunUsage.calls)
        ? {
            runId: chat.lastRunUsage.runId,
            calls: chat.lastRunUsage.calls.flatMap((call) => {
              const usage = normalizeModelCallUsage(call);
              return usage ? [usage] : [];
            }),
          }
        : undefined,
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
    ...(chat.capability !== undefined
      ? { capability: normalizeAgentDiagnosticCapability(chat.capability) }
      : {}),
  };
};

function applyChatAccounting(chat: AgentChat) {
  const attempts = mergeModelAttempts(chat.modelAttempts);
  chat.modelAttempts = attempts;
  chat.legacyUsage ??= {
    totalTokensUsed: chat.totalTokensUsed,
    hasUnreportedUsage: Boolean(
      chat.hasUnreportedUsage || chat.totalTokensUsed > 0
    ),
  };
  Object.assign(chat, projectChatUsage(chat.legacyUsage, attempts));
  if (chat.runBudget) {
    chat.runBudget = projectRunBudget(chat.runBudget, attempts);
    chat.hasUnreportedUsage ||= chat.runBudget.hasUnreportedUsage;
  }
  const runId = chat.lastRunUsage?.runId;
  if (runId && attempts.some((attempt) => attempt.runId === runId)) {
    chat.lastRunUsage = {
      runId,
      calls: attempts
        .filter(
          (attempt) =>
            attempt.runId === runId &&
            attempt.outcome === "response" &&
            attempt.usage
        )
        .sort((a, b) => a.ordinal - b.ordinal)
        .map((attempt) => ({
          ...attempt.usage!,
          attemptId: attempt.attemptId,
        })),
    };
  }
}

function isValidChatCheckpoint(
  chat: AgentChat,
  messages: AgentMessage[],
  checkpoint: AgentRunCheckpoint
) {
  const root = messages.findIndex(
    (message) =>
      message.id === checkpoint.rootUserMessageId && message.role === "user"
  );
  const through = messages.findIndex(
    (message) => message.id === checkpoint.throughMessageId
  );
  return (
    !chat.archived &&
    chat.runId === checkpoint.runId &&
    chat.runBudget?.runId === checkpoint.runId &&
    root >= 0 &&
    through >= root &&
    samePersistedValue(chat.capability, checkpoint.bindings.capability)
  );
}

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
  queuedUserMessagesByChatId: {},
});

const normalizePersistedState = (
  raw: unknown,
  recoverInterruptedRuns = true
): PersistedAgentState => {
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

  // Invalid capabilities must not restore as unrestricted chats.
  const chats = rawChats
    .filter(
      (chat) =>
        !isRecord(chat) ||
        chat.capability === undefined ||
        normalizeAgentDiagnosticCapability(chat.capability) !== undefined
    )
    .map((chat) => normalizeChat(chat));
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
    if (chat.legacyUsage || chat.modelAttempts || chat.runBudget)
      applyChatAccounting(chat);
    if (
      chat.runCheckpoint &&
      !isValidChatCheckpoint(chat, messages, chat.runCheckpoint)
    )
      chat.runCheckpoint = undefined;
    if (
      (chat.status === "paused" || chat.runBudget) &&
      (!chat.runCheckpoint || chat.runBudget?.runId !== chat.runId)
    ) {
      if (chat.runId || chat.status === "paused") {
        chat.status = DEFAULT_CHAT_STATUS;
        chat.interrupted = true;
        chat.interruptionReason = "unexpected";
      }
      chat.runCheckpoint = undefined;
    }

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
    if (recoverInterruptedRuns && chat.status === "running") {
      chat.status = DEFAULT_CHAT_STATUS;
      chat.interrupted = true;
      chat.interruptionReason = "unexpected";
      chat.lastError = null;
      chat.requiresAIConfiguration = false;
      if (chat.runCheckpoint) chat.runCheckpoint.suspension = "interrupted";
    }
    if (chat.interrupted) {
      setLastUserInterruptionReason(
        messages,
        chat.interruptionReason ?? "unexpected"
      );
    }

    const pendingAsk = normalizePendingAsk(
      rawPendingAskByChatId[chat.id],
      getCurrentTurnResourceReferences(messages).at(-1)?.project ?? chat.context
    );
    if (chat.status === "awaiting_user") {
      if (pendingAsk) {
        pendingAskByChatId[chat.id] = pendingAsk;
      } else if (recoverInterruptedRuns) {
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

  const chatIds = new Set(chats.map((chat) => chat.id));
  const queuedUserMessagesByChatId: Record<string, QueuedUserMessage> = {};
  if (isRecord(raw.queuedUserMessagesByChatId)) {
    for (const [chatId, value] of Object.entries(
      raw.queuedUserMessagesByChatId
    )) {
      const message = normalizeQueuedUserMessage(value, chatIds);
      if (message?.chatId === chatId) {
        queuedUserMessagesByChatId[chatId] = message;
      }
    }
  }
  const legacyQueuedMessage = normalizeQueuedUserMessage(
    raw.queuedUserMessage,
    chatIds
  );
  if (legacyQueuedMessage) {
    queuedUserMessagesByChatId[legacyQueuedMessage.chatId] ??=
      legacyQueuedMessage;
  }

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
    queuedUserMessagesByChatId,
    activeChatIds:
      !recoverInterruptedRuns && Array.isArray(raw.activeChatIds)
        ? raw.activeChatIds.filter(
            (id): id is string => typeof id === "string" && chatIds.has(id)
          )
        : undefined,
  };
};

// --- Zustand store types ---

export interface AgentState {
  // Window state
  visible: boolean;
  size: { width: number; height: number };
  sidebarWidth: number;
  minimized: boolean;

  preferences: AgentPreferences;
  preferencesStorageError: boolean;
  updatePreferences: (patch: Partial<AgentPreferences>) => void;
  resetPreferences: () => void;
  syncPreferences: () => void;

  // Chat state
  workspaceName: string | null;
  userEmail: string | null;
  chats: AgentChat[];
  messagesByChatId: Record<string, AgentMessage[]>;
  pendingAskByChatId: Record<string, AgentPendingAsk>;
  draftsByChatId: Record<string, AgentDraft>;
  currentChatId: string | null;
  abortControllersByChatId: Record<string, AbortController>;
  queuedUserMessagesByChatId: Record<string, QueuedUserMessage>;
  queuedUserMessageInFlightByChatId: Record<string, boolean>;
  remoteBusyChatIds: Record<string, boolean>;
  expressionDraftTargetsByChatId: Record<
    string,
    RegisteredExpressionDraftTarget
  >;

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
  registerExpressionDraftTarget: (
    chatId: string,
    target: AgentExpressionDraftTarget,
    options?: { allowRaw?: boolean }
  ) => () => void;
  canApplyExpressionDraft: (
    chatId: string,
    mode: AgentExpressionDraftMode
  ) => boolean;
  applyExpressionDraft: (
    chatId: string,
    expression: string,
    mode: AgentExpressionDraftMode
  ) => Promise<boolean>;

  // Messages
  addMessage: (message: AddMessageOptions) => AgentMessage | null;
  removeMessagesByRunId: (
    chatId: string,
    runId?: string | null
  ) => AgentMessage[];
  appendToolCall: (
    chatId: string,
    messageId: string,
    toolCall: ToolCall
  ) => AgentMessage | null;
  appendContinueActions: (
    chatId: string,
    messageId: string,
    actions: Extract<AgentResponseAction, { kind: "continue" }>[],
    references: AgentResourceReference[]
  ) => AgentMessage | null;
  recordMaskingRuleDraftHandoff: (
    chatId: string,
    messageId: string,
    draftId: string,
    outcome: "opened" | "applied"
  ) => AgentMessage | null;
  recordMaskingExemptionDraftHandoff: AgentState["recordMaskingRuleDraftHandoff"];
  recordSemanticTypeDraftHandoff: (
    chatId: string,
    messageId: string,
    draftId: string,
    outcome: "opened" | "created"
  ) => AgentMessage | null;
  markMaskingExemptionDraftCreated: (
    chatId: string,
    messageId: string,
    draftId: string
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
  beginChatRun: (
    chatId: string,
    rootUserMessageId: string,
    bindings: AgentRunBindings
  ) => Promise<string | null>;
  continueChatRun: (chatId: string, runId: string) => Promise<boolean>;
  saveChatRunCheckpoint: (
    chatId: string,
    checkpoint: AgentRunCheckpoint
  ) => Promise<boolean>;
  reserveChatRunAttempt: (
    chatId: string,
    runId: string
  ) => Promise<AgentAttemptReservation>;
  recordChatModelAttempt: (
    chatId: string,
    attempt: AgentModelAttempt
  ) => Promise<boolean>;
  pauseChatRun: (
    chatId: string,
    checkpoint: AgentRunCheckpoint
  ) => Promise<boolean>;
  increaseRunTokenLimit: (chatId: string, limit: number) => Promise<boolean>;
  endPausedRun: (chatId: string) => Promise<boolean>;
  flushRunPersistence: () => Promise<boolean>;
  canUseAgentExecution: () => boolean;
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
  recordChatModelCallUsage: (
    chatId: string,
    runId: string,
    usage: AgentModelCallUsage
  ) => void;

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
  setWorkspaceName: (workspaceName?: string, userEmail?: string) => boolean;
  saveWindowState: () => void;
  loadWindowState: () => void;
  saveState: () => void;
  loadState: () => void;
  syncState: () => void;

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

export const selectContinueChatForContext = (
  state: AgentState,
  context?: string,
  workspaceName?: string
): AgentChat | null => {
  if (!context || !workspaceName || state.workspaceName !== workspaceName) {
    return null;
  }
  return (
    selectOrderedChats(state).find(
      (chat) =>
        chat.context === context &&
        chat.workspaceName === workspaceName &&
        !chat.archived &&
        state.canSelectChat(chat.id)
    ) ?? null
  );
};

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
  selectCurrentChat(state)?.status === "running" ||
  Boolean(
    state.currentChatId && state.remoteBusyChatIds?.[state.currentChatId]
  );

export const selectError = (state: AgentState): string | null =>
  selectCurrentChat(state)?.lastError ?? null;

export const selectCurrentChatRequiresAIConfiguration = (
  state: AgentState
): boolean => selectCurrentChat(state)?.requiresAIConfiguration ?? false;

export const selectHasRunningChat = (state: AgentState): boolean =>
  state.chats.some((chat) => chat.status === "running");

// --- Store creation ---

const getSafeStorage = (): Storage | null => {
  try {
    const storage = globalThis.localStorage as Partial<Storage> | undefined;
    if (
      !storage ||
      typeof storage.getItem !== "function" ||
      typeof storage.setItem !== "function" ||
      typeof storage.removeItem !== "function"
    ) {
      return null;
    }
    // Browsers may expose Storage while denying access to its contents.
    storage.getItem(AGENT_STATE_KEY);
    return storage as Storage;
  } catch {
    return null;
  }
};

const loadAgentPreferences = (workspaceName: string, userEmail: string) => {
  const key = getAgentPreferencesKey(workspaceName, userEmail);
  try {
    const storage = getSafeStorage();
    if (!storage || !key)
      throw new Error("Agent preference storage unavailable");
    const serialized = storage.getItem(key);
    return {
      preferences: normalizeAgentPreferences(
        serialized ? JSON.parse(serialized) : null
      ),
      preferencesStorageError: false,
    };
  } catch {
    return {
      preferences: { ...DEFAULT_AGENT_PREFERENCES },
      preferencesStorageError: true,
    };
  }
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
    if (!key.startsWith(`${AGENT_RUN_STATE_KEY}.`)) storage?.removeItem(key);
    return null;
  }
};

const getPersistedState = (storage: Storage | null, key: string) =>
  ensureCurrentChat(
    readPersistedState(storage, key) ?? createEmptyPersistedState()
  );

const selectPersistedChats = (
  state: PersistedAgentState,
  chats: AgentChat[]
): PersistedAgentState => {
  const ids = new Set(chats.map((chat) => chat.id));
  return {
    currentChatId: ids.has(state.currentChatId ?? "")
      ? state.currentChatId
      : (chats[0]?.id ?? null),
    chats,
    messagesByChatId: Object.fromEntries(
      chats.map((chat) => [chat.id, state.messagesByChatId[chat.id] ?? []])
    ),
    pendingAskByChatId: Object.fromEntries(
      chats.flatMap((chat) => {
        const ask = state.pendingAskByChatId[chat.id];
        return ask ? [[chat.id, ask]] : [];
      })
    ),
    draftsByChatId: Object.fromEntries(
      chats.flatMap((chat) => {
        const draft = state.draftsByChatId[chat.id];
        return draft ? [[chat.id, draft]] : [];
      })
    ),
    queuedUserMessagesByChatId: Object.fromEntries(
      chats.flatMap((chat) => {
        const message = state.queuedUserMessagesByChatId[chat.id];
        return message ? [[chat.id, message]] : [];
      })
    ),
  };
};

const mergePersistedStates = (
  existing: PersistedAgentState | null,
  incoming: PersistedAgentState
): PersistedAgentState => {
  if (!existing) return incoming;
  const byId = new Map(
    incoming.chats.map((chat) => [chat.id, { chat, state: incoming }])
  );
  for (const chat of existing.chats) {
    const previous = byId.get(chat.id);
    if (!previous || previous.chat.updatedTs <= chat.updatedTs) {
      byId.set(chat.id, { chat, state: existing });
    }
  }
  const chats = [...byId.values()].map(({ chat }) => chat);
  const merged: PersistedAgentState = {
    currentChatId:
      existing.currentChatId && byId.has(existing.currentChatId)
        ? existing.currentChatId
        : incoming.currentChatId,
    chats,
    messagesByChatId: {},
    pendingAskByChatId: {},
    draftsByChatId: {},
    queuedUserMessagesByChatId: {},
  };
  for (const [id, { state }] of byId) {
    merged.messagesByChatId[id] = state.messagesByChatId[id] ?? [];
    if (state.pendingAskByChatId[id]) {
      merged.pendingAskByChatId[id] = state.pendingAskByChatId[id];
    }
    if (state.draftsByChatId[id]) {
      merged.draftsByChatId[id] = state.draftsByChatId[id];
    }
    if (state.queuedUserMessagesByChatId[id]) {
      merged.queuedUserMessagesByChatId[id] =
        state.queuedUserMessagesByChatId[id];
    }
  }
  return merged;
};

const hasOnlyEmptyChat = (state: PersistedAgentState | null) => {
  if (!state || state.chats.length === 0) return true;
  if (
    state.chats.length !== 1 ||
    Object.keys(state.queuedUserMessagesByChatId).length > 0
  )
    return false;
  const chat = state.chats[0];
  return (
    !state.messagesByChatId[chat.id]?.length &&
    !state.pendingAskByChatId[chat.id] &&
    !state.draftsByChatId[chat.id]
  );
};

const isScopedAgentStateKey = (key: string) => {
  const suffix = key.slice(`${AGENT_STATE_KEY}.`.length);
  const separator = suffix.indexOf(":");
  if (separator < 1) return false;
  const workspaceIdLength = Number(suffix.slice(0, separator));
  if (!Number.isSafeInteger(workspaceIdLength) || workspaceIdLength < 1)
    return false;
  const remainder = suffix.slice(separator + 1);
  return (
    remainder[workspaceIdLength] === "." &&
    remainder.slice(workspaceIdLength + 1).includes("@")
  );
};

const migrateLegacyState = (
  storage: Storage | null,
  workspaceName: string,
  userEmail: string
) => {
  if (!storage || !getAgentStateKey(workspaceName, userEmail)) return;
  try {
    const prefix = `${AGENT_STATE_KEY}.`;
    const keys = Array.from({ length: storage.length }, (_, index) =>
      storage.key(index)
    ).filter((key): key is string => Boolean(key));
    const sources = keys.flatMap((key) => {
      const globalSource =
        key === AGENT_STATE_KEY || key === LEGACY_AGENT_STATE_KEY;
      if (
        !globalSource &&
        (!key.startsWith(prefix) || isScopedAgentStateKey(key))
      ) {
        return [];
      }
      const saved = storage.getItem(key);
      if (!saved) return [];
      try {
        const raw = JSON.parse(saved);
        if (
          !isRecord(raw) ||
          (!Array.isArray(raw.chats) && !Array.isArray(raw.threads))
        ) {
          return [];
        }
        return [
          {
            key,
            workspaceName: globalSource
              ? null
              : `${workspaceNamePrefix}${key.slice(prefix.length)}`,
            state: normalizePersistedState(raw),
          },
        ];
      } catch {
        return [];
      }
    });
    if (sources.length === 0) return;

    const owner = storage.getItem(LEGACY_AGENT_OWNER_KEY);
    if (owner !== null && owner !== userEmail) return;
    if (owner === null) storage.setItem(LEGACY_AGENT_OWNER_KEY, userEmail);

    const incomingByWorkspace = new Map<string, PersistedAgentState>();
    for (const source of sources) {
      const chatsByWorkspace = new Map<string, AgentChat[]>();
      for (const chat of source.state.chats) {
        const chatWorkspaceName =
          source.workspaceName ??
          (chat.workspaceName?.startsWith(workspaceNamePrefix)
            ? chat.workspaceName
            : chat.context?.startsWith(workspaceNamePrefix)
              ? chat.context
              : workspaceName);
        const chats = chatsByWorkspace.get(chatWorkspaceName) ?? [];
        chats.push({ ...chat, workspaceName: chatWorkspaceName });
        chatsByWorkspace.set(chatWorkspaceName, chats);
      }
      for (const [name, chats] of chatsByWorkspace) {
        const incoming = selectPersistedChats(source.state, chats);
        incomingByWorkspace.set(
          name,
          mergePersistedStates(incomingByWorkspace.get(name) ?? null, incoming)
        );
      }
    }

    // Write every account cache before removing any ownerless source.
    for (const [name, incoming] of incomingByWorkspace) {
      const key = getAgentStateKey(name, userEmail);
      if (!key) throw new Error("Invalid legacy workspace");
      const existing = readPersistedState(storage, key);
      const merged = mergePersistedStates(existing, incoming);
      if (hasOnlyEmptyChat(existing)) {
        merged.currentChatId = incoming.currentChatId;
      }
      storage.setItem(key, JSON.stringify(merged));
    }
    for (const { key } of sources) storage.removeItem(key);
  } catch {
    // The owner marker prevents another account from claiming a partial migration.
  }
};

const loadInitialState = (): {
  chats: AgentChat[];
  messagesByChatId: Record<string, AgentMessage[]>;
  pendingAskByChatId: Record<string, AgentPendingAsk>;
  draftsByChatId: Record<string, AgentDraft>;
  queuedUserMessagesByChatId: Record<string, QueuedUserMessage>;
  currentChatId: string | null;
} => {
  return ensureCurrentChat(createEmptyPersistedState());
};

const persistedSnapshot = (state: AgentState): PersistedAgentState => ({
  currentChatId: state.currentChatId,
  chats: state.chats,
  messagesByChatId: state.messagesByChatId,
  pendingAskByChatId: state.pendingAskByChatId,
  draftsByChatId: state.draftsByChatId,
  queuedUserMessagesByChatId: state.queuedUserMessagesByChatId,
  activeChatIds: Object.keys(state.abortControllersByChatId),
});

const samePersistedValue = (a: unknown, b: unknown) =>
  a === b || JSON.stringify(a) === JSON.stringify(b);

const currentScopeKey = (state: AgentState) =>
  state.workspaceName && state.userEmail
    ? getAgentStateKey(state.workspaceName, state.userEmail)
    : null;
const canOwnRun = (state: AgentState, chatId: string) => {
  const key = currentScopeKey(state);
  if (!key) return false;
  if (supportsAgentRunRecovery()) return hasAgentRunLease(key, chatId);
  try {
    return !getSafeStorage()?.getItem(
      key.replace(AGENT_STATE_KEY, AGENT_RUN_STATE_KEY)
    );
  } catch {
    return false;
  }
};

const draftShadowKey = (scopeKey: string) => `bb-agent-ui-v1.${scopeKey}`;
const saveDraftShadow = (state: AgentState) => {
  const key = currentScopeKey(state);
  if (!key || !supportsAgentRunRecovery()) return;
  try {
    sessionStorage.setItem(
      draftShadowKey(key),
      JSON.stringify({
        currentChatId: state.currentChatId,
        draftsByChatId: state.draftsByChatId,
      })
    );
  } catch {
    /* In-memory drafts remain available when browser storage is unavailable. */
  }
};
const restoreDraftShadow = (key: string, state: PersistedAgentState) => {
  if (!supportsAgentRunRecovery()) return state;
  try {
    const raw = JSON.parse(
      sessionStorage.getItem(draftShadowKey(key)) ?? "null"
    );
    if (!isRecord(raw)) return state;
    const ids = new Set(state.chats.map((chat) => chat.id));
    if (
      raw.currentChatId === null ||
      (typeof raw.currentChatId === "string" && ids.has(raw.currentChatId))
    )
      state.currentChatId = raw.currentChatId as string | null;
    if (isRecord(raw.draftsByChatId))
      for (const id of ids) {
        const draft = normalizeDraft(raw.draftsByChatId[id]);
        if (draft) state.draftsByChatId[id] = draft;
        else delete state.draftsByChatId[id];
      }
  } catch {
    /* Draft recovery never restores execution state. */
  }
  return state;
};

const mergeDraftHandoffs = (
  message: AgentMessage,
  other: AgentMessage | undefined
): AgentMessage => {
  if (!other || other.role !== "assistant" || other.chatId !== message.chatId)
    return message;
  let metadata = message.metadata;
  for (const [key, kind] of [
    ["maskingRuleDraftHandoff", "draft-masking-rule"],
    ["maskingExemptionDraftHandoff", "draft-masking-exemption"],
    ["semanticTypeDraftHandoff", "draft-semantic-type"],
  ] as const) {
    const incoming = normalizeDraftHandoff(
      other.metadata?.[key],
      kind,
      message.role,
      message.metadata?.actions ?? []
    );
    if (!incoming) continue;
    const current = normalizeDraftHandoff(
      metadata?.[key],
      kind,
      message.role,
      message.metadata?.actions ?? []
    );
    if (current && current.draftId !== incoming.draftId) continue;
    const receipt = current
      ? { ...current, ...(incoming.created ? { created: true as const } : {}) }
      : incoming;
    if (!samePersistedValue(metadata?.[key], receipt))
      metadata = { ...metadata, [key]: receipt };
  }
  return metadata === message.metadata ? message : { ...message, metadata };
};

const chatExecution = (state: PersistedAgentState, chat: AgentChat) => {
  const {
    title: _title,
    context: _context,
    workspaceName: _workspaceName,
    createdTs: _createdTs,
    updatedTs: _updatedTs,
    page: _page,
    archived: _archived,
    totalTokensUsed: _total,
    hasUnreportedUsage: _unreported,
    modelAttempts: _attempts,
    legacyUsage: _legacy,
    lastRunUsage: _usage,
    runBudget: budget,
    ...execution
  } = normalizeChat(chat);
  return {
    ...execution,
    ...(!chat.runBudget && !chat.modelAttempts
      ? {
          totalTokensUsed: chat.totalTokensUsed,
          hasUnreportedUsage: chat.hasUnreportedUsage,
          lastRunUsage: chat.lastRunUsage,
        }
      : {}),
    runBudget: budget
      ? {
          ...budget,
          totalTokensUsed: 0,
          hasUnreportedUsage: false,
          limit: 0,
          revision: 0,
        }
      : undefined,
    messages: (state.messagesByChatId[chat.id] ?? []).map((message) => {
      if (!message.metadata) return message;
      const {
        maskingRuleDraftHandoff: _ruleReceipt,
        maskingExemptionDraftHandoff: _exemptionReceipt,
        semanticTypeDraftHandoff: _typeReceipt,
        ...metadata
      } = message.metadata;
      return { ...message, metadata };
    }),
    pendingAsk: state.pendingAskByChatId[chat.id],
    queuedMessage: state.queuedUserMessagesByChatId[chat.id],
    active: state.activeChatIds?.includes(chat.id) ?? false,
  };
};

// Rebase only this tab's edits; an untouched snapshot must not undo another tab.
const rebasePersistedChanges = (
  base: PersistedAgentState,
  local: PersistedAgentState,
  remote: PersistedAgentState,
  locallyBusy: Set<string>
): PersistedAgentState => {
  const baseChats = new Map(base.chats.map((chat) => [chat.id, chat]));
  const localChats = new Map(local.chats.map((chat) => [chat.id, chat]));
  const remoteChats = new Map(remote.chats.map((chat) => [chat.id, chat]));
  const result = createEmptyPersistedState();
  result.activeChatIds = [];
  for (const id of new Set([...remoteChats.keys(), ...localChats.keys()])) {
    const previous = baseChats.get(id);
    const current = localChats.get(id);
    const incoming = remoteChats.get(id);
    if (previous && (!current || (!incoming && !locallyBusy.has(id)))) continue;
    const useLocal =
      current &&
      (!previous ||
        locallyBusy.has(id) ||
        !samePersistedValue(
          chatExecution(base, previous),
          chatExecution(local, current)
        ));
    const source = useLocal ? local : remote;
    const sourceChat = useLocal ? current : incoming;
    if (!sourceChat) continue;
    const chat = { ...sourceChat };
    if (
      current?.modelAttempts ||
      incoming?.modelAttempts ||
      current?.legacyUsage ||
      incoming?.legacyUsage
    ) {
      chat.legacyUsage = incoming?.legacyUsage ??
        current?.legacyUsage ?? {
          totalTokensUsed: sourceChat.totalTokensUsed,
          hasUnreportedUsage: Boolean(
            sourceChat.hasUnreportedUsage || sourceChat.totalTokensUsed > 0
          ),
        };
      if (current?.legacyUsage && incoming?.legacyUsage) {
        chat.legacyUsage = {
          totalTokensUsed: Math.max(
            current.legacyUsage.totalTokensUsed,
            incoming.legacyUsage.totalTokensUsed
          ),
          hasUnreportedUsage:
            current.legacyUsage.hasUnreportedUsage ||
            incoming.legacyUsage.hasUnreportedUsage,
        };
      }
      chat.modelAttempts = mergeModelAttempts(
        incoming?.modelAttempts,
        current?.modelAttempts
      );
      if (
        chat.runBudget &&
        current?.runBudget?.runId === chat.runBudget.runId &&
        incoming?.runBudget?.runId === chat.runBudget.runId
      ) {
        chat.runBudget = {
          ...chat.runBudget,
          limit: Math.max(current.runBudget.limit, incoming.runBudget.limit),
          revision: Math.max(
            current.runBudget.revision,
            incoming.runBudget.revision
          ),
        };
      }
      applyChatAccounting(chat);
    }
    if (current?.capability || incoming?.capability) {
      chat.capability = current?.capability ?? incoming?.capability;
    }
    if (current && incoming && previous) {
      for (const key of [
        "title",
        "context",
        "workspaceName",
        "page",
        "archived",
      ] as const) {
        if (!samePersistedValue(current[key], previous[key])) {
          Object.assign(chat, { [key]: current[key] });
        } else {
          Object.assign(chat, { [key]: incoming[key] });
        }
      }
      chat.updatedTs =
        current.updatedTs === previous.updatedTs
          ? incoming.updatedTs
          : incoming.updatedTs === previous.updatedTs
            ? current.updatedTs
            : Math.max(current.updatedTs, incoming.updatedTs);
    }
    result.chats.push(chat);
    // Native forms may finish in another tab while this tab owns execution.
    const otherMessages = new Map(
      (useLocal ? remote : local).messagesByChatId[id]?.map((message) => [
        message.id,
        message,
      ])
    );
    result.messagesByChatId[id] = (source.messagesByChatId[id] ?? []).map(
      (message) => mergeDraftHandoffs(message, otherMessages.get(message.id))
    );
    if (source.activeChatIds?.includes(id)) result.activeChatIds.push(id);
    if (source.pendingAskByChatId[id])
      result.pendingAskByChatId[id] = source.pendingAskByChatId[id];
    if (source.queuedUserMessagesByChatId[id])
      result.queuedUserMessagesByChatId[id] =
        source.queuedUserMessagesByChatId[id];
    const draft = samePersistedValue(
      local.draftsByChatId[id],
      base.draftsByChatId[id]
    )
      ? remote.draftsByChatId[id]
      : local.draftsByChatId[id];
    if (draft) result.draftsByChatId[id] = draft;
  }
  const selected =
    local.currentChatId !== base.currentChatId
      ? local.currentChatId
      : remote.currentChatId;
  result.currentChatId =
    selected && result.chats.some((chat) => chat.id === selected)
      ? selected
      : null;
  return result;
};

export const createAgentStore = () => {
  const initial = loadInitialState();
  const initialWindowState = getInitialWindowState();
  const lastSerializedByScope = new Map<string, string>();
  const snapshotsByScope = new Map<string, PersistedAgentState>();
  const storageSnapshotsByScope = new Map<string, PersistedAgentState>();
  const pendingStatesByScope = new Map<string, AgentState>();
  const writersByScope = new Map<
    string,
    ReturnType<typeof createAgentRunPersistence>
  >();
  const getWriter = (key: string) => {
    let writer = writersByScope.get(key);
    if (!writer) {
      writer = createAgentRunPersistence(key, () => {
        const state = pendingStatesByScope.get(key);
        if (!state || !synchronizeState(state, true, true))
          throw new Error("Agent persistence unavailable");
      });
      writersByScope.set(key, writer);
    }
    return writer;
  };
  let applyingSnapshot = false;
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

      const recordMaskingDraftHandoff = (
        kind: "draft-masking-rule" | "draft-masking-exemption",
        key: "maskingRuleDraftHandoff" | "maskingExemptionDraftHandoff",
        chatId: string,
        messageId: string,
        draftId: string,
        outcome: "opened" | "applied"
      ): AgentMessage | null => {
        get().syncState();
        const state = get();
        const message = state.messagesByChatId[chatId]?.find(
          (candidate) => candidate.id === messageId
        );
        if (
          !state.workspaceName ||
          !state.userEmail ||
          _getChat(chatId)?.workspaceName !== state.workspaceName ||
          message?.chatId !== chatId ||
          message.role !== "assistant" ||
          !message.metadata?.actions?.some(
            (action) => action.kind === kind && action.draftId === draftId
          ) ||
          (outcome !== "opened" && outcome !== "applied")
        )
          return null;
        const receipt = message.metadata[key];
        if (receipt) return receipt.draftId === draftId ? message : null;
        set((draft) => {
          const current = draft.messagesByChatId[chatId]?.find(
            (candidate) => candidate.id === messageId
          );
          if (current?.metadata) current.metadata[key] = { draftId, outcome };
        });
        return (
          get().messagesByChatId[chatId]?.find(
            (candidate) => candidate.id === messageId
          ) ?? null
        );
      };

      return {
        // Window state
        visible: false,
        size: initialWindowState.size,
        sidebarWidth: initialWindowState.sidebarWidth,
        minimized: false,

        preferences: { ...DEFAULT_AGENT_PREFERENCES },
        preferencesStorageError: false,
        updatePreferences: (patch) => {
          const { workspaceName, userEmail } = get();
          const key =
            workspaceName && userEmail
              ? getAgentPreferencesKey(workspaceName, userEmail)
              : null;
          if (!key) return;
          const preferences = normalizeAgentPreferences({
            ...get().preferences,
            ...patch,
          });
          let preferencesStorageError = false;
          try {
            const storage = getSafeStorage();
            if (!storage)
              throw new Error("Agent preference storage unavailable");
            storage.setItem(key, JSON.stringify(preferences));
          } catch {
            preferencesStorageError = true;
          }
          set((draft) => {
            draft.preferences = preferences;
            draft.preferencesStorageError = preferencesStorageError;
          });
        },
        resetPreferences: () =>
          get().updatePreferences(DEFAULT_AGENT_PREFERENCES),
        syncPreferences: () => {
          const { workspaceName, userEmail } = get();
          if (!workspaceName || !userEmail) return;
          set(loadAgentPreferences(workspaceName, userEmail));
        },

        // Chat state (from persisted/initial)
        workspaceName: null,
        userEmail: null,
        chats: initial.chats,
        messagesByChatId: initial.messagesByChatId,
        pendingAskByChatId: initial.pendingAskByChatId,
        draftsByChatId: initial.draftsByChatId,
        currentChatId: initial.currentChatId,
        abortControllersByChatId: {},
        queuedUserMessagesByChatId: initial.queuedUserMessagesByChatId,
        queuedUserMessageInFlightByChatId: {},
        remoteBusyChatIds: {},
        expressionDraftTargetsByChatId: {},

        // UI actions
        open: () =>
          set((state) => {
            state.visible = true;
            state.minimized = false;
          }),
        toggle: () => {
          if (!get().visible || get().minimized) {
            get().open();
            return;
          }
          set((state) => {
            state.visible = false;
          });
        },
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

        isChatRunning: (chatId) =>
          _getChat(chatId)?.status === "running" ||
          Boolean(chatId && get().remoteBusyChatIds[chatId]),

        canSelectChat: (chatId) => {
          return Boolean(chatId && _getChat(chatId));
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
          set((state) => {
            state.chats.push(chat);
            state.messagesByChatId[chat.id] = [];
            if (options.select ?? true) {
              state.currentChatId = chat.id;
            }
          });
          return chat;
        },

        queueUserMessage: (message) => {
          get().syncState();
          if (
            get().remoteBusyChatIds[message.chatId] ||
            _getChat(message.chatId)?.status !== "idle" ||
            _getChat(message.chatId)?.runCheckpoint?.suspension ===
              "token-limit" ||
            get().queuedUserMessagesByChatId[message.chatId]
          ) {
            return;
          }
          const { resourceReference, resourceReferences, ...rest } = message;
          const normalizedReferences = normalizeAgentResourceReferences(
            resourceReferences,
            resourceReference
          );
          set((state) => {
            state.queuedUserMessagesByChatId[message.chatId] = {
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
            delete state.queuedUserMessageInFlightByChatId[message.chatId];
          });
        },

        takeQueuedUserMessage: (chatId) => {
          const message = get().queuedUserMessagesByChatId[chatId];
          if (
            !message ||
            _getChat(chatId)?.runCheckpoint?.suspension === "token-limit" ||
            get().remoteBusyChatIds[chatId] ||
            get().queuedUserMessageInFlightByChatId[chatId]
          ) {
            return null;
          }
          set((state) => {
            state.queuedUserMessageInFlightByChatId[chatId] = true;
          });
          return message;
        },

        finishQueuedUserMessage: (chatId) =>
          set((state) => {
            delete state.queuedUserMessagesByChatId[chatId];
            delete state.queuedUserMessageInFlightByChatId[chatId];
            delete state.expressionDraftTargetsByChatId[chatId];
          }),

        commitQueuedUserMessage: (chatId) => {
          const queued = get().queuedUserMessagesByChatId[chatId];
          if (
            !queued ||
            get().remoteBusyChatIds[chatId] ||
            !_getChat(chatId) ||
            _getChat(chatId)?.runCheckpoint?.suspension === "token-limit"
          ) {
            return null;
          }

          let message: AgentMessage | null = null;
          set((state) => {
            const queued = state.queuedUserMessagesByChatId[chatId];
            if (!queued) return;
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
            state.queuedUserMessageInFlightByChatId[chatId] = true;
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
          const current = _getChat(get().currentChatId);
          if (!current)
            return get().createChat({ page, context, workspaceName });
          if (
            current.context === context &&
            current.workspaceName === workspaceName
          ) {
            return current;
          }
          if (
            current.status === "running" ||
            get().getPendingAsk(current.id) ||
            get().queuedUserMessagesByChatId[current.id]
          ) {
            return get().createChat({ page, context, workspaceName });
          }
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
          if (!context) {
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
            chat.capability === undefined &&
            chat.status === "idle" &&
            !get().getPendingAsk(chat.id) &&
            !get().queuedUserMessagesByChatId[chat.id];
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
          if (_getChat(chatId)?.runBudget)
            get().getAbortController(chatId)?.abort();
          set((state) => {
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              chat.archived = true;
              if (chat.runBudget) {
                chat.runCheckpoint = undefined;
                chat.runId = null;
                delete state.pendingAskByChatId[chatId];
                delete state.queuedUserMessagesByChatId[chatId];
                delete state.queuedUserMessageInFlightByChatId[chatId];
                if (
                  chat.status === "running" ||
                  chat.status === "paused" ||
                  chat.status === "awaiting_user"
                ) {
                  chat.status = "idle";
                  chat.interrupted = true;
                }
              }
              _touchChatInDraft(state.chats, chatId);
            }
          });
          return _getChat(chatId);
        },

        archiveAllActiveChats: () => {
          let archivedCount = 0;
          for (const chat of get().chats)
            if (!chat.archived && chat.runBudget)
              get().getAbortController(chat.id)?.abort();
          set((state) => {
            const activeChats = state.chats.filter((chat) => !chat.archived);
            archivedCount = activeChats.length;
            if (archivedCount === 0) return;

            for (const chat of activeChats) {
              chat.archived = true;
              if (chat.runBudget) {
                chat.runCheckpoint = undefined;
                chat.runId = null;
                delete state.pendingAskByChatId[chat.id];
                delete state.queuedUserMessagesByChatId[chat.id];
                delete state.queuedUserMessageInFlightByChatId[chat.id];
                if (
                  chat.status === "running" ||
                  chat.status === "paused" ||
                  chat.status === "awaiting_user"
                ) {
                  chat.status = "idle";
                  chat.interrupted = true;
                }
              }
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
          get().syncState();
          if (!_getChat(chatId) || get().remoteBusyChatIds[chatId])
            return false;
          get().cancel(chatId);
          set((state) => {
            state.chats = state.chats.filter((c) => c.id !== chatId);
            delete state.messagesByChatId[chatId];
            delete state.pendingAskByChatId[chatId];
            delete state.draftsByChatId[chatId];
            delete state.abortControllersByChatId[chatId];
            delete state.queuedUserMessagesByChatId[chatId];
            delete state.queuedUserMessageInFlightByChatId[chatId];
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
          get().syncState();
          const archivedChatIds = get()
            .chats.filter(
              (chat) => chat.archived && !get().remoteBusyChatIds[chat.id]
            )
            .map((chat) => chat.id);
          for (const chatId of archivedChatIds) {
            get().cancel(chatId);
          }

          let deletedCount = 0;
          set((state) => {
            const archivedChats = state.chats.filter(
              (chat) => chat.archived && !state.remoteBusyChatIds[chat.id]
            );
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

            for (const chatId of deletedChatIds) {
              delete state.queuedUserMessagesByChatId[chatId];
              delete state.queuedUserMessageInFlightByChatId[chatId];
              delete state.expressionDraftTargetsByChatId[chatId];
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

        registerExpressionDraftTarget: (chatId, apply, options = {}) => {
          const registered: RegisteredExpressionDraftTarget = {
            apply,
            allowRaw: options.allowRaw ?? true,
            token: Symbol(),
          };
          set((state) => {
            state.expressionDraftTargetsByChatId[chatId] = registered;
          });
          return () =>
            set((state) => {
              if (
                state.expressionDraftTargetsByChatId[chatId]?.token ===
                registered.token
              ) {
                delete state.expressionDraftTargetsByChatId[chatId];
              }
            });
        },

        canApplyExpressionDraft: (chatId, mode) => {
          const target = get().expressionDraftTargetsByChatId[chatId];
          return Boolean(target && (mode === "selector" || target.allowRaw));
        },

        applyExpressionDraft: async (chatId, expression, mode) => {
          const target = get().expressionDraftTargetsByChatId[chatId];
          if (!target || (mode === "raw" && !target.allowRaw)) return false;
          return target.apply(expression, mode);
        },

        // Messages
        addMessage: (message) => {
          if (message.role === "user") get().syncState();
          if (message.chatId && !_getChat(message.chatId)) return null;
          const chatId = message.chatId ?? get().ensureCurrentChat().id;
          if (get().remoteBusyChatIds[chatId]) return null;
          if (
            message.role === "user" &&
            (_getChat(chatId)?.status === "paused" ||
              _getChat(chatId)?.runCheckpoint?.suspension === "token-limit")
          )
            return null;
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

        appendContinueActions: (chatId, messageId, newActions, references) => {
          const messages = get().messagesByChatId[chatId];
          const message = messages?.at(-1);
          if (
            !_getChat(chatId) ||
            message?.id !== messageId ||
            message.role !== "assistant"
          ) {
            return null;
          }
          const existingActions = message.metadata?.actions ?? [];
          const actions = normalizeAgentResponseActions([
            ...existingActions,
            ...newActions,
          ]);
          if (actions.length <= existingActions.length) return null;
          set((state) => {
            const current = state.messagesByChatId[chatId]?.at(-1);
            if (current?.id !== messageId) return;
            current.metadata = {
              ...current.metadata,
              actions,
              ...(references.length > 0
                ? { resourceReferences: references }
                : {}),
            };
          });
          return get().messagesByChatId[chatId]?.at(-1) ?? null;
        },

        recordSemanticTypeDraftHandoff: (
          chatId,
          messageId,
          draftId,
          outcome
        ) => {
          get().syncState();
          const state = get();
          const message = state.messagesByChatId[chatId]?.find(
            (candidate) => candidate.id === messageId
          );
          const receipt = message?.metadata?.semanticTypeDraftHandoff;
          if (
            !state.workspaceName ||
            !state.userEmail ||
            _getChat(chatId)?.workspaceName !== state.workspaceName ||
            message?.chatId !== chatId ||
            message.role !== "assistant" ||
            !message.metadata?.actions?.some(
              (action) =>
                action.kind === "draft-semantic-type" &&
                action.draftId === draftId
            ) ||
            (outcome !== "opened" && outcome !== "created") ||
            (receipt && receipt.draftId !== draftId) ||
            (outcome === "created" && receipt?.outcome !== "opened")
          )
            return null;
          set((draft) => {
            const current = draft.messagesByChatId[chatId]?.find(
              (candidate) => candidate.id === messageId
            );
            if (current?.metadata)
              current.metadata.semanticTypeDraftHandoff = {
                draftId,
                outcome: "opened",
                ...(receipt?.created || outcome === "created"
                  ? { created: true }
                  : {}),
              };
          });
          return (
            get().messagesByChatId[chatId]?.find(
              (candidate) => candidate.id === messageId
            ) ?? null
          );
        },
        recordMaskingRuleDraftHandoff: (chatId, messageId, draftId, outcome) =>
          recordMaskingDraftHandoff(
            "draft-masking-rule",
            "maskingRuleDraftHandoff",
            chatId,
            messageId,
            draftId,
            outcome
          ),
        recordMaskingExemptionDraftHandoff: (
          chatId,
          messageId,
          draftId,
          outcome
        ) =>
          recordMaskingDraftHandoff(
            "draft-masking-exemption",
            "maskingExemptionDraftHandoff",
            chatId,
            messageId,
            draftId,
            outcome
          ),
        markMaskingExemptionDraftCreated: (chatId, messageId, draftId) => {
          get().syncState();
          const state = get();
          const message = state.messagesByChatId[chatId]?.find(
            (candidate) => candidate.id === messageId
          );
          const receipt = message?.metadata?.maskingExemptionDraftHandoff;
          if (
            !state.workspaceName ||
            !state.userEmail ||
            _getChat(chatId)?.workspaceName !== state.workspaceName ||
            message?.chatId !== chatId ||
            message.role !== "assistant" ||
            !message.metadata?.actions?.some(
              (action) =>
                action.kind === "draft-masking-exemption" &&
                action.draftId === draftId
            ) ||
            receipt?.draftId !== draftId ||
            (receipt.outcome !== "opened" && receipt.outcome !== "applied")
          )
            return null;
          if (receipt.created) return message;
          set((draft) => {
            const current = draft.messagesByChatId[chatId]?.find(
              (candidate) => candidate.id === messageId
            );
            if (current?.metadata?.maskingExemptionDraftHandoff)
              current.metadata.maskingExemptionDraftHandoff.created = true;
          });
          return (
            get().messagesByChatId[chatId]?.find(
              (candidate) => candidate.id === messageId
            ) ?? null
          );
        },

        // Status
        setChatStatus: (chatId, status, options = {}) => {
          if (!_getChat(chatId) || get().remoteBusyChatIds[chatId]) return null;
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
                state.queuedUserMessagesByChatId[chatId] &&
                state.messagesByChatId[chatId]?.some(
                  (message) =>
                    message.id ===
                      state.queuedUserMessagesByChatId[chatId]?.id &&
                    message.role === "user"
                )
              ) {
                delete state.queuedUserMessagesByChatId[chatId];
                delete state.queuedUserMessageInFlightByChatId[chatId];
              }
              _touchChatInDraft(state.chats, chatId);
            }
          });
          return _getChat(chatId);
        },

        startChatRun: (chatId, page, options = {}) => {
          if (
            (currentScopeKey(get()) && !get().canUseAgentExecution()) ||
            _getChat(chatId)?.runCheckpoint ||
            (_getChat(chatId)?.runId && _getChat(chatId)?.runBudget) ||
            _getChat(chatId)?.status === "paused"
          )
            return;
          get().setChatStatus(chatId, "running", { page });
          set((state) => {
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              chat.runId = options.runId ?? chat.runId ?? null;
              chat.runMode = options.runMode;
              if (chat.runId && chat.lastRunUsage?.runId !== chat.runId) {
                chat.lastRunUsage = { runId: chat.runId, calls: [] };
              }
              setLastUserInterruptionReason(
                state.messagesByChatId[chatId] ?? []
              );
            }
          });
        },

        beginChatRun: async (chatId, rootUserMessageId, rawBindings) => {
          get().syncState();
          const chat = _getChat(chatId);
          const bindings = normalizeRunBindings(rawBindings);
          if (
            !chat ||
            chat.archived ||
            chat.runCheckpoint ||
            (chat.runId && chat.runBudget) ||
            chat.status === "paused" ||
            (get().remoteBusyChatIds[chatId] && !supportsAgentRunRecovery()) ||
            !bindings ||
            !samePersistedValue(chat.capability, bindings.capability) ||
            !get()
              .getMessages(chatId)
              .some(
                (message) =>
                  message.id === rootUserMessageId && message.role === "user"
              ) ||
            !canOwnRun(get(), chatId)
          )
            return null;
          const runId = uuidv4();
          set((state) => {
            const current = state.chats.find(
              (candidate) => candidate.id === chatId
            )!;
            current.legacyUsage ??= {
              totalTokensUsed: current.totalTokensUsed,
              hasUnreportedUsage: Boolean(
                current.hasUnreportedUsage || current.totalTokensUsed > 0
              ),
            };
            current.legacyUsage.hasUnreportedUsage ||= Boolean(
              current.runBudget?.hasUnreportedUsage
            );
            current.modelAttempts ??= [];
            current.runId = runId;
            current.runMode = bindings.runMode;
            current.runBudget = createRunBudget(
              runId,
              state.preferences.tokenLimitPerRun
            );
            current.runCheckpoint = {
              version: 1,
              runId,
              suspension: "interrupted",
              rootUserMessageId,
              throughMessageId: rootUserMessageId,
              bindings,
              maskingSections: [],
              phase: "working",
              startupComplete: false,
            };
            current.lastRunUsage = { runId, calls: [] };
            current.status = "running";
            delete state.remoteBusyChatIds[chatId];
            current.interrupted = false;
            current.interruptionReason = undefined;
            current.lastError = null;
            current.requiresAIConfiguration = false;
            delete state.pendingAskByChatId[chatId];
            const root = state.messagesByChatId[chatId]?.find(
              (message) => message.id === rootUserMessageId
            );
            if (root) root.metadata = { ...root.metadata, runId };
            setLastUserInterruptionReason(state.messagesByChatId[chatId] ?? []);
            _touchChatInDraft(state.chats, chatId);
          });
          return (await get().flushRunPersistence()) ? runId : null;
        },

        continueChatRun: async (chatId, runId) => {
          get().syncState();
          const chat = _getChat(chatId);
          const pendingAsk = get().getPendingAsk(chatId);
          if (
            pendingAsk &&
            !get()
              .getMessages(chatId)
              .some(
                (message) =>
                  message.role === "tool" &&
                  message.toolCallId === pendingAsk.toolCallId &&
                  message.metadata?.runId === runId
              )
          )
            return false;
          if (
            !chat ||
            chat.runId !== runId ||
            !chat.runCheckpoint ||
            chat.runCheckpoint.startupComplete !== true ||
            !isValidChatCheckpoint(
              chat,
              get().getMessages(chatId),
              chat.runCheckpoint
            ) ||
            (get().remoteBusyChatIds[chatId] && !supportsAgentRunRecovery()) ||
            !canOwnRun(get(), chatId)
          )
            return false;
          set((state) => {
            const current = state.chats.find(
              (candidate) => candidate.id === chatId
            )!;
            current.status = "running";
            delete state.remoteBusyChatIds[chatId];
            current.interrupted = false;
            current.interruptionReason = undefined;
            current.lastError = null;
            current.requiresAIConfiguration = false;
            delete state.pendingAskByChatId[chatId];
            setLastUserInterruptionReason(state.messagesByChatId[chatId] ?? []);
            current.runCheckpoint!.suspension = "interrupted";
          });
          return get().flushRunPersistence();
        },

        saveChatRunCheckpoint: async (chatId, rawCheckpoint) => {
          const checkpoint = normalizeRunCheckpoint(rawCheckpoint);
          const chat = _getChat(chatId);
          if (
            !chat ||
            !checkpoint ||
            !canOwnRun(get(), chatId) ||
            !isValidChatCheckpoint(
              chat,
              get().getMessages(chatId),
              checkpoint
            ) ||
            !chat.runCheckpoint ||
            !samePersistedValue(
              chat.runCheckpoint.bindings,
              checkpoint.bindings
            ) ||
            chat.runCheckpoint.rootUserMessageId !==
              checkpoint.rootUserMessageId ||
            (chat.runCheckpoint.phase === "finalizing" &&
              checkpoint.phase !== "finalizing") ||
            (chat.runCheckpoint.startupComplete && !checkpoint.startupComplete)
          )
            return false;
          set((state) => {
            state.chats.find(
              (candidate) => candidate.id === chatId
            )!.runCheckpoint = checkpoint;
          });
          return get().flushRunPersistence();
        },

        reserveChatRunAttempt: async (chatId, runId) => {
          get().syncState();
          const chat = _getChat(chatId);
          if (
            !chat ||
            chat.archived ||
            chat.runId !== runId ||
            chat.status !== "running" ||
            !chat.runBudget ||
            !chat.runCheckpoint ||
            !canOwnRun(get(), chatId)
          )
            return { kind: "superseded" };
          if (!(await get().flushRunPersistence()))
            return { kind: "storage-unavailable" };
          const latest = _getChat(chatId);
          if (
            !latest?.runBudget ||
            latest.runId !== runId ||
            !canOwnRun(get(), chatId)
          )
            return { kind: "superseded" };
          const result = reserveRunAttempt(latest.runBudget);
          if (result.reservation.kind !== "allowed") return result.reservation;
          set((state) => {
            const current = state.chats.find(
              (candidate) => candidate.id === chatId
            )!;
            current.runBudget = projectRunBudget(
              result.budget,
              current.modelAttempts ?? []
            );
            current.hasUnreportedUsage = true;
          });
          return (await get().flushRunPersistence())
            ? result.reservation
            : { kind: "storage-unavailable" };
        },

        recordChatModelAttempt: async (chatId, rawAttempt) => {
          const attempt = normalizeModelAttempt(rawAttempt);
          const chat = _getChat(chatId);
          if (
            !chat ||
            !attempt ||
            !get()
              .getMessages(chatId)
              .some((message) => message.metadata?.runId === attempt.runId)
          )
            return false;
          const existing = chat.modelAttempts?.find(
            (record) => record.attemptId === attempt.attemptId
          );
          if (
            chat.runBudget?.runId === attempt.runId &&
            attempt.ordinal > chat.runBudget.providerAttemptCount
          )
            return false;
          if (!existing) {
            set((state) => {
              const current = state.chats.find(
                (candidate) => candidate.id === chatId
              )!;
              current.modelAttempts = mergeModelAttempts(
                current.modelAttempts,
                [attempt]
              );
              applyChatAccounting(current);
            });
          }
          if (!(await get().flushRunPersistence())) {
            throw new Error("Agent attempt could not be persisted");
          }
          return true;
        },

        pauseChatRun: async (chatId, checkpoint) => {
          if (
            !(await get().saveChatRunCheckpoint(chatId, {
              ...checkpoint,
              suspension: "token-limit",
            }))
          )
            return false;
          set((state) => {
            const chat = state.chats.find(
              (candidate) => candidate.id === chatId
            );
            if (chat?.runId === checkpoint.runId) {
              chat.status = "paused";
              chat.interrupted = false;
            }
          });
          return get().flushRunPersistence();
        },

        increaseRunTokenLimit: async (chatId, limit) => {
          get().syncState();
          const chat = _getChat(chatId);
          const recoverySupported = supportsAgentRunRecovery();
          const ownsRun = recoverySupported && canOwnRun(get(), chatId);
          const budget =
            chat?.runBudget && increaseRunBudgetLimit(chat.runBudget, limit);
          if (
            !chat ||
            chat.archived ||
            !chat.runCheckpoint ||
            !budget ||
            (recoverySupported && !ownsRun) ||
            (get().remoteBusyChatIds[chatId] && !ownsRun)
          )
            return false;
          set((state) => {
            if (ownsRun) delete state.remoteBusyChatIds[chatId];
            state.chats.find(
              (candidate) => candidate.id === chatId
            )!.runBudget = budget;
          });
          return get().flushRunPersistence();
        },

        endPausedRun: async (chatId) => {
          get().syncState();
          const chat = _getChat(chatId);
          const recoverySupported = supportsAgentRunRecovery();
          const ownsRun = recoverySupported && canOwnRun(get(), chatId);
          if (
            !chat ||
            (!chat.runCheckpoint && !(chat.runId && chat.runBudget)) ||
            (recoverySupported && !ownsRun) ||
            (chat.status === "running" &&
              (!ownsRun || get().getAbortController(chatId))) ||
            (get().remoteBusyChatIds[chatId] && !ownsRun)
          )
            return false;
          set((state) => {
            const current = state.chats.find(
              (candidate) => candidate.id === chatId
            )!;
            current.status = "idle";
            current.runId = null;
            current.runMode = undefined;
            current.runCheckpoint = undefined;
            current.interrupted = false;
            current.interruptionReason = undefined;
            current.lastError = null;
            if (ownsRun) delete state.remoteBusyChatIds[chatId];
            delete state.pendingAskByChatId[chatId];
            delete state.queuedUserMessagesByChatId[chatId];
          });
          return get().flushRunPersistence();
        },

        flushRunPersistence: async () => {
          const state = get();
          const key =
            state.workspaceName && state.userEmail
              ? getAgentStateKey(state.workspaceName, state.userEmail)
              : null;
          if (!key || !getSafeStorage()) return false;
          if (!supportsAgentRunRecovery()) return synchronizeState(state, true);
          pendingStatesByScope.set(key, state);
          const writer = getWriter(key);
          writer.schedule();
          return writer.flush();
        },

        canUseAgentExecution: () => {
          const key = currentScopeKey(get());
          if (!key) return false;
          if (supportsAgentRunRecovery()) return true;
          try {
            return !getSafeStorage()?.getItem(
              key.replace(AGENT_STATE_KEY, AGENT_RUN_STATE_KEY)
            );
          } catch {
            return false;
          }
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
              chat.runCheckpoint = undefined;
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
            const chat = state.chats.find(
              (candidate) => candidate.id === chatId
            );
            if (chat?.runCheckpoint)
              chat.runCheckpoint.suspension = "interrupted";
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
              state.queuedUserMessagesByChatId[chatId] &&
              state.messagesByChatId[chatId]?.some(
                (message) =>
                  message.id === state.queuedUserMessagesByChatId[chatId]?.id &&
                  message.role === "user"
              )
            ) {
              delete state.queuedUserMessagesByChatId[chatId];
              delete state.queuedUserMessageInFlightByChatId[chatId];
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
          get().syncState();
          const id = chatId ?? get().currentChatId;
          const chat = _getChat(id);
          const pendingAsk = id ? get().pendingAskByChatId[id] : undefined;
          if (
            !id ||
            get().remoteBusyChatIds[id] ||
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
            currentChat.runCheckpoint = undefined;
            currentChat.interrupted = false;
            currentChat.interruptionReason = undefined;
            currentChat.lastError = null;
            setLastUserInterruptionReason(state.messagesByChatId[id] ?? []);
            _touchChatInDraft(state.chats, id);
          });
          return true;
        },

        answerPendingAsk: (chatId, response, metadata) => {
          get().syncState();
          if (get().remoteBusyChatIds[chatId]) return null;
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
              if (chat.legacyUsage)
                chat.legacyUsage.totalTokensUsed += totalTokensUsed;
              _touchChatInDraft(state.chats, chatId);
            }
          });
          return _getChat(chatId);
        },

        recordChatModelCallUsage: (chatId, runId, rawUsage) => {
          const usage = normalizeModelCallUsage(rawUsage);
          if (!usage || !_getChat(chatId)) return;
          set((state) => {
            const chat = state.chats.find(
              (candidate) => candidate.id === chatId
            );
            if (!chat) return;
            chat.totalTokensUsed += usage.totalTokens ?? 0;
            if (usage.totalTokens === undefined) chat.hasUnreportedUsage = true;
            if (chat.legacyUsage) {
              chat.legacyUsage.totalTokensUsed += usage.totalTokens ?? 0;
              chat.legacyUsage.hasUnreportedUsage ||=
                usage.totalTokens === undefined;
            }
            if (chat.lastRunUsage?.runId === runId) {
              chat.lastRunUsage.calls.push(usage);
            }
            _touchChatInDraft(state.chats, chatId);
          });
        },

        // Abort / cancel
        cancel: (chatId, reason) => {
          get().syncState();
          const id = chatId ?? get().currentChatId;
          if (!id || get().remoteBusyChatIds[id]) return;
          get().getAbortController(id)?.abort();
          get().setAbortController(id, null);
          if (get().isChatRunning(id)) {
            get().interruptChatRun(id, undefined, reason);
          }
        },

        // Error
        clearError: (chatId) => {
          get().syncState();
          const id = chatId ?? get().currentChatId;
          const chat = _getChat(id);
          if (!chat || get().remoteBusyChatIds[chat.id]) return;
          set((state) => {
            const c = state.chats.find((c) => c.id === id);
            if (c) {
              c.lastError = null;
              c.requiresAIConfiguration = false;
              c.interrupted = false;
              c.interruptionReason = undefined;
              if (!c.runCheckpoint) c.runId = null;
              if (c.status === "error") {
                c.status = DEFAULT_CHAT_STATUS;
              }
              _touchChatInDraft(state.chats, c.id);
            }
          });
        },

        // Persistence
        setWorkspaceName: (workspaceName, userEmail) => {
          const nextUserEmail = userEmail ?? get().userEmail ?? "";
          const stateKey = workspaceName
            ? getAgentStateKey(workspaceName, nextUserEmail)
            : null;
          const identityChanged = get().userEmail !== nextUserEmail;
          if (!workspaceName || !stateKey) {
            if (!get().workspaceName && !get().userEmail) return false;
            get().saveState();
            for (const controller of Object.values(
              get().abortControllersByChatId
            )) {
              controller.abort();
            }
            const empty = ensureCurrentChat(createEmptyPersistedState());
            set((draft) => {
              draft.workspaceName = null;
              draft.userEmail = null;
              draft.chats = empty.chats;
              draft.messagesByChatId = empty.messagesByChatId;
              draft.pendingAskByChatId = empty.pendingAskByChatId;
              draft.draftsByChatId = empty.draftsByChatId;
              draft.currentChatId = empty.currentChatId;
              draft.abortControllersByChatId = {};
              draft.queuedUserMessagesByChatId = {};
              draft.queuedUserMessageInFlightByChatId = {};
              draft.remoteBusyChatIds = {};
              draft.expressionDraftTargetsByChatId = {};
              draft.preferences = { ...DEFAULT_AGENT_PREFERENCES };
              draft.preferencesStorageError = false;
            });
            return false;
          }
          if (
            get().workspaceName === workspaceName &&
            get().userEmail === nextUserEmail
          ) {
            return true;
          }
          if (
            !identityChanged &&
            (_hasRunningChat() ||
              Object.keys(get().queuedUserMessagesByChatId).length > 0)
          ) {
            return false;
          }

          get().saveState();
          if (identityChanged) {
            for (const controller of Object.values(
              get().abortControllersByChatId
            )) {
              controller.abort();
            }
          }
          const storage = getSafeStorage();
          if (!supportsAgentRunRecovery())
            migrateLegacyState(storage, workspaceName, nextUserEmail);
          const oldKey = stateKey.replace(AGENT_RUN_STATE_KEY, AGENT_STATE_KEY);
          const v3Key = stateKey.replace(AGENT_STATE_KEY, AGENT_RUN_STATE_KEY);
          const existingV3 = storage?.getItem(v3Key);
          const serialized = storage?.getItem(stateKey);
          const sourceKey = supportsAgentRunRecovery()
            ? serialized
              ? stateKey
              : oldKey
            : existingV3
              ? v3Key
              : stateKey;
          const persistedState = restoreDraftShadow(
            stateKey,
            getPersistedState(storage, sourceKey)
          );
          if (supportsAgentRunRecovery() && !serialized) {
            for (const chat of persistedState.chats) {
              chat.legacyUsage ??= {
                totalTokensUsed: chat.totalTokensUsed,
                hasUnreportedUsage: Boolean(
                  chat.hasUnreportedUsage ||
                    chat.totalTokensUsed > 0 ||
                    (persistedState.messagesByChatId[chat.id]?.length ?? 0) > 0
                ),
              };
              chat.modelAttempts ??= [];
              applyChatAccounting(chat);
            }
          }
          const preferencesState = loadAgentPreferences(
            workspaceName,
            nextUserEmail
          );
          snapshotsByScope.set(
            stateKey,
            serialized ? persistedState : createEmptyPersistedState()
          );
          storageSnapshotsByScope.set(
            stateKey,
            serialized ? persistedState : createEmptyPersistedState()
          );
          if (serialized) lastSerializedByScope.set(stateKey, serialized);
          set((draft) => {
            draft.workspaceName = workspaceName;
            draft.userEmail = nextUserEmail;
            draft.chats = persistedState.chats;
            draft.messagesByChatId = persistedState.messagesByChatId;
            draft.pendingAskByChatId = persistedState.pendingAskByChatId;
            draft.draftsByChatId = persistedState.draftsByChatId;
            draft.currentChatId = persistedState.currentChatId;
            draft.abortControllersByChatId = {};
            draft.queuedUserMessagesByChatId =
              persistedState.queuedUserMessagesByChatId;
            draft.queuedUserMessageInFlightByChatId = {};
            draft.remoteBusyChatIds = {};
            draft.expressionDraftTargetsByChatId = {};
            draft.preferences = preferencesState.preferences;
            draft.preferencesStorageError =
              preferencesState.preferencesStorageError;
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
          synchronizeState(get(), true);
        },

        syncState: () => synchronizeState(get(), false),

        loadState: () => {
          const storage = getSafeStorage();
          const workspaceName = get().workspaceName;
          const userEmail = get().userEmail;
          const key =
            workspaceName && userEmail
              ? getAgentStateKey(workspaceName, userEmail)
              : null;
          if (!key) return;
          const state = getPersistedState(storage, key);
          snapshotsByScope.set(key, state);
          storageSnapshotsByScope.set(key, state);
          const serialized = storage?.getItem(key);
          if (serialized) lastSerializedByScope.set(key, serialized);
          set((draft) => {
            draft.chats = state.chats;
            draft.messagesByChatId = state.messagesByChatId;
            draft.pendingAskByChatId = state.pendingAskByChatId;
            draft.draftsByChatId = state.draftsByChatId;
            draft.currentChatId = state.currentChatId;
            draft.queuedUserMessagesByChatId = state.queuedUserMessagesByChatId;
            draft.queuedUserMessageInFlightByChatId = {};
            draft.remoteBusyChatIds = {};
          });
        },
      };
    })
  );

  const synchronizeState = (
    state: AgentState,
    persist: boolean,
    underLock = false
  ): boolean => {
    const storage = getSafeStorage();
    const workspaceName = state.workspaceName;
    const userEmail = state.userEmail;
    const key =
      workspaceName && userEmail
        ? getAgentStateKey(workspaceName, userEmail)
        : null;
    if (!storage || !key) {
      return false;
    }
    if (persist && supportsAgentRunRecovery() && !underLock) {
      pendingStatesByScope.set(key, state);
      getWriter(key).schedule();
      return true;
    }
    let saved: string | null;
    try {
      if (
        !supportsAgentRunRecovery() &&
        storage.getItem(key.replace(AGENT_STATE_KEY, AGENT_RUN_STATE_KEY))
      )
        return false;
      saved = storage.getItem(key);
    } catch {
      return false;
    }
    let remote = storageSnapshotsByScope.get(key);
    if (!remote || saved !== lastSerializedByScope.get(key)) {
      try {
        remote = saved
          ? normalizePersistedState(JSON.parse(saved), false)
          : createEmptyPersistedState();
      } catch {
        return false;
      }
    }
    const local = persistedSnapshot(state);
    const base = snapshotsByScope.get(key) ?? createEmptyPersistedState();
    const baseChats = new Map(base.chats.map((chat) => [chat.id, chat]));
    const remoteChats = new Map(remote.chats.map((chat) => [chat.id, chat]));
    const locallyBusy = new Set(
      state.chats
        .filter((chat) => {
          if (
            state.remoteBusyChatIds[chat.id] ||
            !(
              state.abortControllersByChatId[chat.id] ||
              chat.status === "running" ||
              chat.status === "awaiting_user" ||
              state.pendingAskByChatId[chat.id] ||
              state.queuedUserMessagesByChatId[chat.id]
            )
          )
            return false;
          if (state.abortControllersByChatId[chat.id]) return true;
          const previous = baseChats.get(chat.id);
          if (!previous) return true;
          const previousExecution = chatExecution(base, previous);
          if (
            !samePersistedValue(previousExecution, chatExecution(local, chat))
          )
            return true;
          // Restored pending work must not override progress made by its owner.
          const incoming = remoteChats.get(chat.id);
          return Boolean(
            incoming &&
              samePersistedValue(
                previousExecution,
                chatExecution(remote, incoming)
              )
          );
        })
        .map((chat) => chat.id)
    );
    const merged = rebasePersistedChanges(base, local, remote, locallyBusy);
    const serialized = persist ? JSON.stringify(merged) : saved;
    if (persist && serialized !== saved) {
      try {
        storage.setItem(key, serialized!);
      } catch {
        return false;
      }
    }
    if (serialized) lastSerializedByScope.set(key, serialized);
    storageSnapshotsByScope.set(key, persist ? merged : remote);
    if (currentScopeKey(store.getState()) !== key) {
      snapshotsByScope.set(key, merged);
      return true;
    }

    const ids = new Set(merged.chats.map((chat) => chat.id));
    const currentChatId =
      state.currentChatId && !ids.has(state.currentChatId)
        ? (merged.chats.find((chat) => !chat.archived)?.id ??
          merged.chats[0]?.id ??
          null)
        : state.currentChatId;
    const draftsByChatId = Object.fromEntries(
      Object.entries(state.draftsByChatId).filter(([id]) => ids.has(id))
    );
    const remoteBusyChatIds = Object.fromEntries(
      merged.chats
        .filter(
          (chat) =>
            !locallyBusy.has(chat.id) &&
            (chat.status === "running" ||
              chat.status === "awaiting_user" ||
              merged.activeChatIds?.includes(chat.id) ||
              merged.queuedUserMessagesByChatId[chat.id])
        )
        .map((chat) => [chat.id, true])
    );
    applyingSnapshot = true;
    try {
      store.setState((draft) => {
        if (!samePersistedValue(draft.chats, merged.chats))
          draft.chats = merged.chats;
        if (
          !samePersistedValue(draft.messagesByChatId, merged.messagesByChatId)
        )
          draft.messagesByChatId = merged.messagesByChatId;
        if (
          !samePersistedValue(
            draft.pendingAskByChatId,
            merged.pendingAskByChatId
          )
        )
          draft.pendingAskByChatId = merged.pendingAskByChatId;
        if (
          !samePersistedValue(
            draft.queuedUserMessagesByChatId,
            merged.queuedUserMessagesByChatId
          )
        )
          draft.queuedUserMessagesByChatId = merged.queuedUserMessagesByChatId;
        if (!samePersistedValue(draft.draftsByChatId, draftsByChatId))
          draft.draftsByChatId = draftsByChatId;
        if (!samePersistedValue(draft.remoteBusyChatIds, remoteBusyChatIds))
          draft.remoteBusyChatIds = remoteBusyChatIds;
        draft.currentChatId = currentChatId;
      });
      if (!persist && supportsAgentRunRecovery()) {
        snapshotsByScope.set(key, remote);
      } else
        snapshotsByScope.set(key, {
          ...persistedSnapshot(store.getState()),
          draftsByChatId: persist
            ? store.getState().draftsByChatId
            : base.draftsByChatId,
        });
    } finally {
      applyingSnapshot = false;
    }
    return true;
  };

  store.subscribe((state, previousState) => {
    if (applyingSnapshot) return;
    const nonDraftStateChanged =
      state.workspaceName !== previousState.workspaceName ||
      state.userEmail !== previousState.userEmail ||
      state.currentChatId !== previousState.currentChatId ||
      state.chats !== previousState.chats ||
      state.messagesByChatId !== previousState.messagesByChatId ||
      state.pendingAskByChatId !== previousState.pendingAskByChatId ||
      state.abortControllersByChatId !==
        previousState.abortControllersByChatId ||
      state.queuedUserMessagesByChatId !==
        previousState.queuedUserMessagesByChatId;
    const draftChanged = state.draftsByChatId !== previousState.draftsByChatId;
    if (!nonDraftStateChanged && !draftChanged) return;
    saveDraftShadow(state);

    clearDraftPersistTimer();
    if (!nonDraftStateChanged && draftChanged) {
      draftPersistTimer = setTimeout(() => {
        draftPersistTimer = null;
        synchronizeState(store.getState(), true);
      }, DRAFT_PERSIST_DELAY_MS);
    } else {
      synchronizeState(state, true);
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
  window.addEventListener("storage", (event) => {
    const { workspaceName, userEmail } = useAgentStore.getState();
    if (
      !workspaceName ||
      !userEmail ||
      (event.storageArea && event.storageArea !== getSafeStorage())
    )
      return;
    if (
      event.key === null ||
      event.key === getAgentStateKey(workspaceName, userEmail)
    ) {
      useAgentStore.getState().syncState();
    }
    if (
      event.key === null ||
      event.key === getAgentPreferencesKey(workspaceName, userEmail)
    ) {
      useAgentStore.getState().syncPreferences();
    }
  });
  window.addEventListener("focus", () => useAgentStore.getState().syncState());
  window.addEventListener("pagehide", () => {
    useAgentStore.getState().syncState();
    const state = useAgentStore.getState();
    for (const chat of state.chats) {
      if (state.remoteBusyChatIds[chat.id]) continue;
      const controller = state.abortControllersByChatId[chat.id];
      const queued = state.queuedUserMessagesByChatId[chat.id];
      controller?.abort();
      // Keep the question for an explicit retry, never replay an approval on exit.
      if (queued) state.commitQueuedUserMessage(chat.id);
      if (
        chat.status === "running" ||
        chat.status === "awaiting_user" ||
        queued
      )
        state.interruptChatRun(chat.id);
      if (queued) state.finishQueuedUserMessage(chat.id);
      if (controller) state.setAbortController(chat.id, null);
    }
    useAgentStore.getState().saveState();
  });
}
