import {
  type RefObject,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { router } from "@/app/router";
import { Button } from "@/components/ui/button";
import { projectNamePrefix } from "@/lib/resourceName";
import { useAppStore } from "@/stores/app";
import {
  getNameParentTokens,
  issueNamePrefix,
} from "@/stores/modules/v1/common";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { runAgentLoop } from "../logic/agentLoop";
import {
  getAgentAIConfigurationEnabled,
  isAgentAIConfigurationError,
} from "../logic/aiConfiguration";
import { getCurrentChatContext } from "../logic/chatContext";
import type { IssueDecisionImpact } from "../logic/issueDecisionImpact";
import {
  getLandingStarterIdForText,
  type LandingStarterId,
} from "../logic/landingStarters";
import { buildOutboundHistory } from "../logic/outboundHistory";
import { buildSystemPrompt } from "../logic/prompt";
import {
  canAddResourceMention,
  getDatabaseMentionLabel,
  getDraftResourceMentions,
  getMentionTriggerRange,
  hasUnboundResourceMention,
  insertResourceMention,
  normalizeAgentResourceReferences,
  removeMentionTrigger,
  updateDraftResourceMentions,
} from "../logic/resourceMention";
import { createToolExecutor, getToolDefinitions } from "../logic/tools";
import { extractDatabaseOverviewEvidence } from "../logic/tools/databaseOverview";
import { extractIssueOverviewEvidence } from "../logic/tools/issueOverview";
import {
  getPendingApiCallPreview,
  resolvePendingApiCall,
} from "../logic/tools/pendingApiCall";
import { createProjectOpenIssuesTool } from "../logic/tools/projectOpenIssues";
import { extractProjectOverviewEvidence } from "../logic/tools/projectOverview";
import { getRecentActivity } from "../logic/tools/recentActivity";
import { createWorkspaceAttentionTool } from "../logic/tools/workspaceAttention";
import { extractWorkspaceOverviewEvidence } from "../logic/tools/workspaceOverview";
import type {
  AgentAnswerEvidence,
  AgentAskUserOption,
  AgentEvidence,
  AgentResourceReference,
  AgentRunMode,
  Message,
} from "../logic/types";
import {
  selectCurrentChat,
  selectCurrentChatRequiresAIConfiguration,
  selectCurrentPendingAsk,
  selectHasRunningChat,
  selectLoading,
  useAgentStore,
} from "../store/agent";
import { AgentContextPicker } from "./AgentContextPicker";
import { AgentMentionTextarea } from "./AgentMentionTextarea";
import { AgentResourcePicker } from "./AgentResourcePicker";

// Module-level map — survives across renders.
const runTokens = new Map<string, number>();

const tokenFormatter = new Intl.NumberFormat();

const formatPermissionTime = (expiresAt: number): string => {
  const seconds = Math.max(0, Math.ceil((expiresAt - Date.now()) / 1000));
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}`;
};

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

const getCurrentPageSnapshot = () => ({
  path: router.currentRoute.value.fullPath,
  title: document.title,
});

interface RecentActivityEvent {
  actor: string;
  action: string;
  resource: string;
  occurredAt?: string;
}

interface RecentActivityResult {
  events: RecentActivityEvent[];
  partial: boolean;
  error?: string;
}

interface AttentionResult {
  issues: {
    name: string;
    title: string;
    relation: "review" | "created";
  }[];
  partial: boolean;
  error?: string;
}

const parseRecentActivityResult = (value: string): RecentActivityResult => {
  try {
    const parsed = JSON.parse(value) as Record<string, unknown>;
    if (typeof parsed.error === "string") {
      return { events: [], partial: false, error: parsed.error };
    }
    const events = Array.isArray(parsed.events)
      ? parsed.events.flatMap((event) => {
          if (
            typeof event !== "object" ||
            event === null ||
            typeof event.actor !== "string" ||
            typeof event.action !== "string" ||
            typeof event.resource !== "string"
          ) {
            return [];
          }
          return [
            {
              actor: event.actor,
              action: event.action,
              resource: event.resource,
              ...(typeof event.occurredAt === "string"
                ? { occurredAt: event.occurredAt }
                : {}),
            },
          ];
        })
      : [];
    return { events, partial: parsed.partial === true };
  } catch {
    return { events: [], partial: false, error: "invalid result" };
  }
};

const parseAttentionResult = (value: string): AttentionResult => {
  try {
    const parsed = JSON.parse(value) as Record<string, unknown>;
    if (typeof parsed.error === "string") {
      return { issues: [], partial: false, error: parsed.error };
    }
    const issues = Array.isArray(parsed.issues)
      ? parsed.issues.flatMap((issue) => {
          if (
            typeof issue !== "object" ||
            issue === null ||
            typeof issue.name !== "string" ||
            typeof issue.title !== "string"
          ) {
            return [];
          }
          return [
            {
              name: issue.name,
              title: issue.title,
              relation:
                issue.relation === "created"
                  ? ("created" as const)
                  : ("review" as const),
            },
          ];
        })
      : [];
    return { issues, partial: parsed.partial === true };
  } catch {
    return { issues: [], partial: false, error: "invalid result" };
  }
};

const mergeOverviewEvidence = (
  current: AgentAnswerEvidence | undefined,
  next: AgentEvidence | undefined
): AgentAnswerEvidence | undefined => {
  if (!next || next.references.length === 0) return current;
  if (!current) return next;

  const groups = Array.isArray(current) ? current : [current];
  const matchingIndex = groups.findIndex(
    (group) =>
      group.project === next.project && group.workspace === next.workspace
  );
  if (matchingIndex < 0) return [...groups, next];

  const existing = groups[matchingIndex];
  const seen = new Set(
    existing.references.map(
      (reference) => `${reference.kind}:${reference.name}`
    )
  );
  const references = [...existing.references];
  for (const reference of next.references) {
    const key = `${reference.kind}:${reference.name}`;
    if (seen.has(key)) continue;
    seen.add(key);
    references.push(reference);
  }
  const merged = { ...existing, references };
  if (!Array.isArray(current)) return merged;
  return groups.map((group, index) =>
    index === matchingIndex ? merged : group
  );
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function AgentInput({
  retryActionRef,
}: {
  retryActionRef?: RefObject<(() => void) | null>;
}) {
  const { t, i18n } = useTranslation();
  // Zustand selectors
  const currentChat = useAgentStore(selectCurrentChat);
  const currentPendingAsk = useAgentStore(selectCurrentPendingAsk);
  const loading = useAgentStore(selectLoading);
  const hasRunningChat = useAgentStore(selectHasRunningChat);
  const isAIConfigurationBlocked = useAgentStore(
    selectCurrentChatRequiresAIConfiguration
  );
  const currentChatId = useAgentStore((s) => s.currentChatId);
  const currentDraft = useAgentStore((s) =>
    currentChatId ? s.draftsByChatId[currentChatId] : undefined
  );
  const queuedUserMessage = useAgentStore((s) => s.queuedUserMessage);
  const getOrFetchSettingByName = useAppStore((s) => s.getOrFetchSettingByName);
  const currentUserName = useAppStore((s) => s.currentUser?.name ?? "");
  const context = currentChat?.context;
  const projectContext = context?.startsWith(projectNamePrefix)
    ? context
    : undefined;
  const contextProject = useAppStore((s) =>
    projectContext ? s.projectsByName[projectContext] : undefined
  );
  const workspace = useAppStore((s) => s.workspace);
  const getOrFetchProjectByName = useAppStore((s) => s.getOrFetchProjectByName);
  const aiConfigurationEnabled = useAppStore((s) =>
    getAgentAIConfigurationEnabled(s.getSettingByName(Setting_SettingName.AI))
  );

  const formatRecentActivity = useCallback(
    (result: string) => {
      const payload = parseRecentActivityResult(result);
      if (payload.error) {
        return t("agent.recent-activity.unavailable");
      }
      if (payload.events.length === 0) {
        return t("agent.recent-activity.empty");
      }

      const formatTime = (value: string | undefined) => {
        if (!value) return t("agent.recent-activity.unknown-time");
        const timestamp = Date.parse(value);
        if (Number.isNaN(timestamp)) {
          return t("agent.recent-activity.unknown-time");
        }
        return new Intl.DateTimeFormat(i18n.language, {
          month: "short",
          day: "numeric",
          hour: "numeric",
          minute: "2-digit",
        }).format(timestamp);
      };

      return [
        t("agent.recent-activity.summary"),
        ...payload.events.map(
          (event) =>
            `- ${t("agent.recent-activity.event", {
              actor: event.actor,
              action: event.action,
              resource: event.resource,
              time: formatTime(event.occurredAt),
            })}`
        ),
        ...(payload.partial ? [t("agent.recent-activity.more")] : []),
      ].join("\n");
    },
    [i18n.language, t]
  );

  const formatWorkspaceAttention = useCallback(
    (result: string) => {
      const payload = parseAttentionResult(result);
      if (payload.error) {
        return t("agent.workspace-attention.unavailable");
      }
      if (payload.issues.length === 0 && payload.partial) {
        return t("agent.workspace-attention.partial");
      }
      if (payload.issues.length === 0) {
        return t("agent.workspace-attention.empty");
      }
      return [
        t("agent.workspace-attention.summary"),
        ...payload.issues.map((issue) =>
          t(`agent.workspace-attention.issue-${issue.relation}`, {
            title: issue.title,
          })
        ),
        ...(payload.partial ? [t("agent.workspace-attention.more")] : []),
      ].join("\n");
    },
    [t]
  );

  const formatProjectAttention = useCallback(
    (result: string) => {
      const payload = parseAttentionResult(result);
      if (payload.error) {
        return t("agent.project-attention.unavailable");
      }
      if (payload.issues.length === 0 && payload.partial) {
        return t("agent.project-attention.partial");
      }
      if (payload.issues.length === 0) {
        return t("agent.project-attention.empty");
      }
      return [
        t("agent.project-attention.summary"),
        ...payload.issues.map((issue) =>
          t(`agent.project-attention.issue-${issue.relation}`, {
            title: issue.title,
          })
        ),
        ...(payload.partial ? [t("agent.project-attention.more")] : []),
      ].join("\n");
    },
    [t]
  );

  const [askInput, setAskInput] = useState("");
  const [, setPermissionTick] = useState(0);
  const [resourcePickerOpen, setResourcePickerOpen] = useState(false);
  const mentionRangeRef = useRef<{ start: number; end: number } | null>(null);
  const selectedMentionCaretRef = useRef<number | null>(null);
  const previousChatIdRef = useRef<string | null>(null);
  const confirmationInFlightRef = useRef(false);

  const composerRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Derived values
  const isCurrentChatRunning = currentChat?.status === "running";
  const chooseOptions =
    currentPendingAsk?.kind === "choose"
      ? (currentPendingAsk.options ?? [])
      : [];
  const isAwaitingConfirm = currentPendingAsk?.kind === "confirm";
  const apiPermission = currentPendingAsk?.apiPermission;
  const canCancelPendingRequest =
    currentChat?.status === "awaiting_user" &&
    Boolean(currentPendingAsk) &&
    !apiPermission;
  const apiPermissionPreview =
    apiPermission && currentChat && currentUserName
      ? getPendingApiCallPreview({
          actionId: apiPermission.actionId,
          chatId: currentChat.id,
          toolCallId: currentPendingAsk.toolCallId,
          userName: currentUserName,
        })
      : undefined;
  useEffect(() => {
    if (!apiPermissionPreview) return;
    const timer = window.setInterval(
      () => setPermissionTick((tick) => tick + 1),
      1000
    );
    return () => window.clearInterval(timer);
  }, [apiPermissionPreview?.expiresAt]);
  const isAwaitingChoose =
    currentPendingAsk?.kind === "choose" && chooseOptions.length > 0;
  const input = currentPendingAsk ? askInput : (currentDraft?.text ?? "");
  const setInput = useCallback(
    (value: string) => {
      if (currentPendingAsk) {
        setAskInput(value);
        return;
      }
      const store = useAgentStore.getState();
      const chatId =
        store.currentChatId ??
        store.ensureCurrentChat(
          getCurrentPageSnapshot(),
          getCurrentChatContext(),
          workspace?.name
        ).id;
      const draft = store.getDraft(chatId);
      const mentions = updateDraftResourceMentions(
        draft.text,
        value,
        getDraftResourceMentions(draft)
      );
      store.setDraft(chatId, {
        text: value,
        ...(mentions.length > 0 ? { mentions } : {}),
      });
    },
    [currentPendingAsk, workspace?.name]
  );
  const handleResourcePickerOpenChange = useCallback(
    (nextOpen: boolean) => {
      setResourcePickerOpen(nextOpen);
      if (nextOpen) return;
      const trigger = mentionRangeRef.current;
      mentionRangeRef.current = null;
      const removed = removeMentionTrigger(input, trigger);
      if (!removed) return;
      selectedMentionCaretRef.current = removed.caret;
      setInput(removed.text);
    },
    [input, setInput]
  );
  const sendLabel = currentPendingAsk ? t("agent.reply") : t("agent.send");
  const inputPlaceholder =
    currentPendingAsk?.kind === "input"
      ? currentPendingAsk.prompt
      : currentPendingAsk?.kind === "choose"
        ? currentPendingAsk.prompt
        : t("agent.input-placeholder");
  const confirmLabel = currentPendingAsk?.confirmLabel ?? t("agent.confirm");
  const cancelLabel = currentPendingAsk?.cancelLabel ?? t("agent.cancel");
  const draftMentions =
    currentPendingAsk || !currentDraft
      ? []
      : getDraftResourceMentions(currentDraft);
  const selectedProjectMention = draftMentions.find(
    ({ resource }) => resource.kind === "project"
  );
  const hasAdditionalMention =
    !currentPendingAsk &&
    draftMentions.length > 0 &&
    hasUnboundResourceMention(input, draftMentions);
  const isSendDisabled =
    isCurrentChatRunning ||
    isAwaitingConfirm ||
    isAwaitingChoose ||
    isAIConfigurationBlocked ||
    hasAdditionalMention ||
    !input.trim();
  const currentChatTokenUsageLabel = useMemo(
    () =>
      t("agent.chat-total-tokens", {
        count: tokenFormatter.format(currentChat?.totalTokensUsed ?? 0),
      }),
    [currentChat?.totalTokensUsed, t]
  );
  const contextTitle = projectContext
    ? contextProject?.title || projectContext.slice(projectNamePrefix.length)
    : context === workspace?.name
      ? workspace?.title || context
      : context;
  const canMentionResource = Boolean(workspace?.name && !currentPendingAsk);

  const selectContext = useCallback(
    (resource: string) => {
      const store = useAgentStore.getState();
      const selected = store.selectOrCreateChatForContext(
        resource,
        getCurrentPageSnapshot(),
        workspace?.name
      );
      if (!selected) return false;
      setResourcePickerOpen(false);
      mentionRangeRef.current = null;
      return true;
    },
    [workspace?.name]
  );

  const selectMentionResource = useCallback(
    (resource: AgentResourceReference) => {
      const store = useAgentStore.getState();
      const chatId =
        store.currentChatId ??
        store.ensureCurrentChat(
          getCurrentPageSnapshot(),
          getCurrentChatContext(),
          workspace?.name
        ).id;
      const draft = store.getDraft(chatId);
      const mentions = getDraftResourceMentions(draft);
      if (!canAddResourceMention(mentions, resource)) return;
      const range = mentionRangeRef.current;
      const label = resource.label;
      const insertion = insertResourceMention(draft.text, range, label);
      if (!insertion) return;
      selectedMentionCaretRef.current = insertion.caret;
      store.setDraft(chatId, {
        text: insertion.text,
        mentions: [
          ...updateDraftResourceMentions(draft.text, insertion.text, mentions),
          { resource, range: insertion.mention },
        ],
      });
      mentionRangeRef.current = null;
    },
    [workspace?.name]
  );

  const selectMentionProject = useCallback(
    (project: Project) =>
      selectMentionResource({
        kind: "project",
        name: project.name,
        project: project.name,
        label: project.title || project.name,
      }),
    [selectMentionResource]
  );

  const selectMentionDatabase = useCallback(
    (database: Database) =>
      selectMentionResource({
        kind: "database",
        name: database.name,
        project: database.project,
        label: getDatabaseMentionLabel(database),
      }),
    [selectMentionResource]
  );

  const selectMentionIssue = useCallback(
    (issue: Issue) => {
      const [projectId] = getNameParentTokens(issue.name, [
        projectNamePrefix,
        issueNamePrefix,
      ]);
      if (!projectId) return;
      selectMentionResource({
        kind: "issue",
        name: issue.name,
        project: `${projectNamePrefix}${projectId}`,
        label: issue.title || issue.name,
      });
    },
    [selectMentionResource]
  );

  useEffect(() => {
    if (projectContext && !contextProject) {
      void getOrFetchProjectByName(projectContext, true);
    }
  }, [contextProject, getOrFetchProjectByName, projectContext]);

  const blockWhenAIIsNotConfigured = useCallback(
    async (chatId: string) => {
      const setting = await getOrFetchSettingByName(
        Setting_SettingName.AI,
        true
      );
      if (getAgentAIConfigurationEnabled(setting) !== false) return false;

      useAgentStore.getState().setChatStatus(chatId, "error", {
        requiresAIConfiguration: true,
      });
      return true;
    },
    [getOrFetchSettingByName]
  );

  useEffect(() => {
    const chat = useAgentStore.getState().getChat(currentChatId);
    if (!chat) return;

    if (aiConfigurationEnabled === false) {
      useAgentStore.getState().setChatStatus(chat.id, "error", {
        requiresAIConfiguration: true,
      });
      return;
    }

    if (aiConfigurationEnabled === true && chat.requiresAIConfiguration) {
      useAgentStore.getState().clearError(chat.id);
    }
  }, [aiConfigurationEnabled, currentChatId]);

  useEffect(() => {
    const chat = useAgentStore.getState().getChat(currentChatId);
    if (!chat) return;
    void blockWhenAIIsNotConfigured(chat.id);
  }, [blockWhenAIIsNotConfigured, currentChatId]);

  // Pending-ask answers are separate from the saved question draft.
  useEffect(() => {
    if (
      currentPendingAsk?.kind === "input" ||
      currentPendingAsk?.kind === "choose"
    ) {
      setAskInput(currentPendingAsk.defaultValue ?? "");
      return;
    }
    setAskInput("");
  }, [
    currentChatId,
    currentPendingAsk?.toolCallId,
    currentPendingAsk?.kind,
    currentPendingAsk?.defaultValue,
  ]);

  useEffect(() => {
    if (
      previousChatIdRef.current &&
      previousChatIdRef.current !== currentChatId
    ) {
      setResourcePickerOpen(false);
      mentionRangeRef.current = null;
      selectedMentionCaretRef.current = null;
    }
    previousChatIdRef.current = currentChatId;
  }, [currentChatId]);

  useLayoutEffect(() => {
    const caret = selectedMentionCaretRef.current;
    if (caret === null) return;
    selectedMentionCaretRef.current = null;
    textareaRef.current?.focus();
    textareaRef.current?.setSelectionRange(caret, caret);
  }, [input]);

  const autoResize = useCallback(() => {
    const el = textareaRef.current;
    if (!el) return;

    el.style.height = "auto";

    const styles = window.getComputedStyle(el);
    const minHeight =
      Number.parseFloat(styles.minHeight) || el.getBoundingClientRect().height;
    const borderHeight = el.offsetHeight - el.clientHeight;
    const desiredHeight = Math.max(minHeight, el.scrollHeight + borderHeight);
    const composer = composerRef.current;
    const parent = composer?.parentElement;
    const composerChrome = composer
      ? Math.max(0, composer.offsetHeight - el.offsetHeight)
      : 0;
    const availableHeight = parent ? parent.clientHeight - composerChrome : 0;
    const maxHeight =
      availableHeight > 0
        ? Math.max(minHeight, availableHeight)
        : desiredHeight;
    const nextHeight = Math.min(desiredHeight, maxHeight);

    el.style.height = `${nextHeight}px`;
    el.style.overflowY = desiredHeight > maxHeight ? "auto" : "hidden";
  }, []);

  useEffect(() => {
    autoResize();
  }, [input, autoResize]);

  useEffect(() => {
    const composer = composerRef.current;
    const parent = composer?.parentElement;
    if (!composer || !parent || typeof ResizeObserver === "undefined") return;

    const observer = new ResizeObserver(autoResize);
    observer.observe(composer);
    observer.observe(parent);
    return () => observer.disconnect();
  }, [autoResize]);

  // ---------------------------------------------------------------------------
  // Agent loop execution
  // ---------------------------------------------------------------------------

  const buildChatHistory = useCallback(
    (chatId: string, systemPrompt: string): Message[] => {
      return buildOutboundHistory(
        systemPrompt,
        useAgentStore.getState().getMessages(chatId),
        useAgentStore.getState().getChat(chatId)?.context
      );
    },
    []
  );

  const handleOutcome = useCallback(
    (
      chatId: string,
      page: { path: string; title: string },
      errorPrefix: string,
      outcome: Awaited<ReturnType<typeof runAgentLoop>>
    ) => {
      const store = useAgentStore.getState();
      switch (outcome.kind) {
        case "completed":
          store.finishChatRun(chatId);
          return;
        case "awaiting_user":
          store.awaitUser(chatId, outcome.ask);
          return;
        case "aborted":
          store.interruptChatRun(chatId, getCurrentPageSnapshot());
          return;
        case "error": {
          if (!isAgentAIConfigurationError(outcome.error)) {
            store.addMessage({
              chatId,
              role: "assistant",
              content: `${errorPrefix}${outcome.error.message}`,
              metadata: {
                route: page.path,
                error: outcome.error.message,
              },
            });
          }
          store.finishChatRun(chatId, {
            status: "error",
            lastError: outcome.error.message,
            requiresAIConfiguration: isAgentAIConfigurationError(outcome.error),
          });
          return;
        }
      }
    },
    []
  );

  const runChat = useCallback(
    async (
      chatId: string,
      page: { path: string; title: string },
      runId: string,
      runMode: AgentRunMode,
      skipReferencePreflight = false
    ) => {
      const store = useAgentStore.getState();
      const lastUserMessage = store
        .getMessages(chatId)
        .findLast((message) => message.role === "user");
      const references = normalizeAgentResourceReferences(
        lastUserMessage?.metadata?.resourceReferences,
        lastUserMessage?.metadata?.resourceReference
      );
      const projectReference = references.find(
        (reference) => reference.kind === "project"
      );
      const databaseReference = references.find(
        (reference) => reference.kind === "database"
      );
      const issueReference = references.find(
        (reference) => reference.kind === "issue"
      );
      const referencedProject =
        projectReference?.name ??
        databaseReference?.project ??
        issueReference?.project;
      const referencedDatabase = databaseReference?.name;
      const referencedIssue = issueReference?.name;
      const controller = new AbortController();
      const runToken = (runTokens.get(chatId) ?? 0) + 1;
      runTokens.set(chatId, runToken);
      store.setAbortController(chatId, controller);

      const systemPrompt = buildSystemPrompt({
        ...page,
        context: store.getChat(chatId)?.context,
        referencedProject,
        referencedDatabase,
        referencedIssue,
        runMode,
      });
      const tools = getToolDefinitions({ runMode, referencedProject });
      const executor = createToolExecutor(router, {
        chatId,
        userName: useAppStore.getState().currentUser?.name,
        context: store.getChat(chatId)?.context,
        referencedProject,
        referencedDatabase,
        referencedIssue,
        runMode,
        onNavigate: () => {
          useAgentStore
            .getState()
            .updateChatPage(chatId, getCurrentPageSnapshot());
        },
      });
      const toolNamesByCallId = new Map<string, string>();
      let overviewEvidence: AgentAnswerEvidence | undefined;

      try {
        const history = buildChatHistory(chatId, systemPrompt);
        let prefetchedProjectOverview: string | undefined;
        if (
          !skipReferencePreflight &&
          projectReference &&
          !databaseReference &&
          !issueReference &&
          referencedProject
        ) {
          const preflightCallId = uuidv4();
          const result = await executor(
            "get_project_overview",
            {},
            preflightCallId
          ).catch(() => undefined);
          if (
            controller.signal.aborted ||
            runToken !== runTokens.get(chatId) ||
            !useAgentStore.getState().getChat(chatId)
          ) {
            return;
          }
          const overviewResult =
            result?.kind === "tool_result" ? result.result : undefined;
          let valid = false;
          if (overviewResult) {
            try {
              const overview = JSON.parse(overviewResult) as {
                project?: unknown;
                error?: unknown;
              };
              valid = overview.project === referencedProject && !overview.error;
            } catch {
              valid = false;
            }
          }
          if (!valid || !overviewResult) {
            useAgentStore.getState().addMessage({
              chatId,
              role: "assistant",
              content: t("agent.project-reference-unavailable"),
              metadata: { route: page.path, runId },
            });
            useAgentStore.getState().finishChatRun(chatId);
            return;
          }
          prefetchedProjectOverview = overviewResult;
          history.push(
            {
              role: "assistant",
              toolCalls: [
                {
                  id: preflightCallId,
                  name: "get_project_overview",
                  arguments: "{}",
                },
              ],
            },
            {
              role: "tool",
              toolCallId: preflightCallId,
              content: overviewResult,
            }
          );
        } else if (
          !skipReferencePreflight &&
          referencedIssue &&
          referencedProject
        ) {
          const preflightCallId = uuidv4();
          const args = { issueName: referencedIssue };
          const result = await executor(
            "get_issue_overview",
            args,
            preflightCallId
          ).catch(() => undefined);
          if (
            controller.signal.aborted ||
            runToken !== runTokens.get(chatId) ||
            !useAgentStore.getState().getChat(chatId)
          ) {
            return;
          }
          const overviewResult =
            result?.kind === "tool_result" ? result.result : undefined;
          let valid = false;
          if (overviewResult) {
            try {
              const overview = JSON.parse(overviewResult) as {
                project?: unknown;
                issue?: { status?: unknown; name?: unknown };
              };
              valid =
                overview.project === referencedProject &&
                overview.issue?.status === "available" &&
                overview.issue.name === referencedIssue;
            } catch {
              valid = false;
            }
          }
          if (!valid || !overviewResult) {
            useAgentStore.getState().addMessage({
              chatId,
              role: "assistant",
              content: t("agent.issue-reference-unavailable"),
              metadata: { route: page.path, runId },
            });
            useAgentStore.getState().finishChatRun(chatId);
            return;
          }
          overviewEvidence = mergeOverviewEvidence(
            overviewEvidence,
            extractIssueOverviewEvidence(overviewResult)
          );
          history.push(
            {
              role: "assistant",
              toolCalls: [
                {
                  id: preflightCallId,
                  name: "get_issue_overview",
                  arguments: JSON.stringify(args),
                },
              ],
            },
            {
              role: "tool",
              toolCallId: preflightCallId,
              content: overviewResult,
            }
          );
        } else if (
          !skipReferencePreflight &&
          referencedDatabase &&
          referencedProject
        ) {
          const preflightCallId = uuidv4();
          const args = { databaseName: referencedDatabase };
          const result = await executor(
            "get_database_overview",
            args,
            preflightCallId
          ).catch(() => undefined);
          if (
            controller.signal.aborted ||
            runToken !== runTokens.get(chatId) ||
            !useAgentStore.getState().getChat(chatId)
          ) {
            return;
          }
          const overviewResult =
            result?.kind === "tool_result" ? result.result : undefined;
          let valid = false;
          if (overviewResult) {
            try {
              const overview = JSON.parse(overviewResult) as {
                project?: unknown;
                database?: {
                  status?: unknown;
                  name?: unknown;
                  state?: unknown;
                };
              };
              valid =
                overview.project === referencedProject &&
                overview.database?.status === "available" &&
                overview.database.name === referencedDatabase &&
                overview.database.state === "ACTIVE";
            } catch {
              valid = false;
            }
          }
          if (!valid || !overviewResult) {
            useAgentStore.getState().addMessage({
              chatId,
              role: "assistant",
              content: t("agent.database-reference-unavailable"),
              metadata: { route: page.path, runId },
            });
            useAgentStore.getState().finishChatRun(chatId);
            return;
          }
          overviewEvidence = mergeOverviewEvidence(
            overviewEvidence,
            extractDatabaseOverviewEvidence(overviewResult)
          );
          history.push(
            {
              role: "assistant",
              toolCalls: [
                {
                  id: preflightCallId,
                  name: "get_database_overview",
                  arguments: JSON.stringify(args),
                },
              ],
            },
            {
              role: "tool",
              toolCallId: preflightCallId,
              content: overviewResult,
            }
          );
        }
        const outcome = await runAgentLoop(
          history,
          tools,
          (name, args, toolCallId) =>
            name === "get_project_overview" && prefetchedProjectOverview
              ? Promise.resolve({
                  kind: "tool_result" as const,
                  result: prefetchedProjectOverview,
                })
              : executor(name, args, toolCallId),
          {
            onAssistantMessage: (message) => {
              for (const toolCall of message.toolCalls ?? []) {
                toolNamesByCallId.set(toolCall.id, toolCall.name);
              }
              if (!message.content && !message.toolCalls?.length) return;
              useAgentStore.getState().addMessage({
                chatId,
                role: "assistant",
                content: message.content,
                toolCalls: message.toolCalls,
                metadata: { route: page.path, runId },
              });
            },
            onToolResult: (toolCallId, result) => {
              const toolName = toolNamesByCallId.get(toolCallId);
              if (toolName === "get_project_overview") {
                overviewEvidence = mergeOverviewEvidence(
                  overviewEvidence,
                  extractProjectOverviewEvidence(result)
                );
              } else if (toolName === "get_issue_overview") {
                overviewEvidence = mergeOverviewEvidence(
                  overviewEvidence,
                  extractIssueOverviewEvidence(result)
                );
              } else if (toolName === "get_database_overview") {
                overviewEvidence = mergeOverviewEvidence(
                  overviewEvidence,
                  extractDatabaseOverviewEvidence(result)
                );
              } else if (toolName === "get_workspace_overview") {
                overviewEvidence = mergeOverviewEvidence(
                  overviewEvidence,
                  extractWorkspaceOverviewEvidence(result)
                );
              }
              useAgentStore.getState().addMessage({
                chatId,
                role: "tool",
                toolCallId,
                content: result,
                metadata: { route: page.path, runId },
              });
            },
            onText: (text) => {
              if (!text) return;
              useAgentStore.getState().addMessage({
                chatId,
                role: "assistant",
                content: text,
                metadata: {
                  route: page.path,
                  runId,
                  ...(overviewEvidence ? { evidence: overviewEvidence } : {}),
                },
              });
            },
          },
          controller.signal,
          { allowPlainTextCompletion: runMode === "read-only" }
        );

        useAgentStore
          .getState()
          .incrementChatTotalTokens(chatId, outcome.totalTokensUsed ?? 0);

        if (runToken !== runTokens.get(chatId)) return;

        handleOutcome(chatId, page, "Error: ", outcome);
      } finally {
        if (
          useAgentStore.getState().getAbortController(chatId) === controller
        ) {
          useAgentStore.getState().setAbortController(chatId, null);
        }
      }
    },
    [buildChatHistory, handleOutcome, t]
  );

  const runRecentActivityStarter = useCallback(
    async (
      chatId: string,
      page: { path: string; title: string },
      runId: string,
      parent: string
    ) => {
      const store = useAgentStore.getState();
      const controller = new AbortController();
      const runToken = (runTokens.get(chatId) ?? 0) + 1;
      runTokens.set(chatId, runToken);
      store.setAbortController(chatId, controller);
      store.addMessage({
        chatId,
        role: "assistant",
        content: t("agent.recent-activity.checking"),
        metadata: { route: page.path, runId, pending: true },
      });

      try {
        const result = await getRecentActivity(parent);
        if (controller.signal.aborted || runToken !== runTokens.get(chatId)) {
          store.removeMessagesByRunId(chatId, runId);
          return;
        }
        store.removeMessagesByRunId(chatId, runId);
        store.addMessage({
          chatId,
          role: "assistant",
          content: formatRecentActivity(result),
          metadata: { route: page.path, runId },
        });
        store.finishChatRun(chatId);
      } catch {
        if (controller.signal.aborted || runToken !== runTokens.get(chatId)) {
          store.removeMessagesByRunId(chatId, runId);
          return;
        }
        store.removeMessagesByRunId(chatId, runId);
        store.addMessage({
          chatId,
          role: "assistant",
          content: t("agent.recent-activity.unavailable"),
          metadata: { route: page.path, runId },
        });
        store.finishChatRun(chatId);
      } finally {
        if (store.getAbortController(chatId) === controller) {
          store.setAbortController(chatId, null);
        }
      }
    },
    [formatRecentActivity, t]
  );

  const runWorkspaceAttentionStarter = useCallback(
    async (
      chatId: string,
      page: { path: string; title: string },
      runId: string
    ) => {
      const store = useAgentStore.getState();
      const controller = new AbortController();
      const runToken = (runTokens.get(chatId) ?? 0) + 1;
      runTokens.set(chatId, runToken);
      store.setAbortController(chatId, controller);
      store.addMessage({
        chatId,
        role: "assistant",
        content: t("agent.workspace-attention.checking"),
        metadata: { route: page.path, runId, pending: true },
      });

      try {
        const result = await createWorkspaceAttentionTool({
          workspaceName: store.getChat(chatId)?.context,
        })();
        if (controller.signal.aborted || runToken !== runTokens.get(chatId)) {
          store.removeMessagesByRunId(chatId, runId);
          return;
        }
        store.removeMessagesByRunId(chatId, runId);
        store.addMessage({
          chatId,
          role: "assistant",
          content: formatWorkspaceAttention(result),
          metadata: { route: page.path, runId },
        });
        store.finishChatRun(chatId);
      } catch {
        if (controller.signal.aborted || runToken !== runTokens.get(chatId)) {
          store.removeMessagesByRunId(chatId, runId);
          return;
        }
        store.removeMessagesByRunId(chatId, runId);
        store.addMessage({
          chatId,
          role: "assistant",
          content: t("agent.workspace-attention.unavailable"),
          metadata: { route: page.path, runId },
        });
        store.finishChatRun(chatId);
      } finally {
        if (store.getAbortController(chatId) === controller) {
          store.setAbortController(chatId, null);
        }
      }
    },
    [formatWorkspaceAttention, t]
  );

  const runProjectAttentionStarter = useCallback(
    async (
      chatId: string,
      page: { path: string; title: string },
      runId: string
    ) => {
      const store = useAgentStore.getState();
      const controller = new AbortController();
      const runToken = (runTokens.get(chatId) ?? 0) + 1;
      runTokens.set(chatId, runToken);
      store.setAbortController(chatId, controller);
      store.addMessage({
        chatId,
        role: "assistant",
        content: t("agent.project-attention.checking"),
        metadata: { route: page.path, runId, pending: true },
      });

      try {
        const result = await createProjectOpenIssuesTool({
          projectName: store.getChat(chatId)?.context,
        })();
        if (controller.signal.aborted || runToken !== runTokens.get(chatId)) {
          store.removeMessagesByRunId(chatId, runId);
          return;
        }
        store.removeMessagesByRunId(chatId, runId);
        store.addMessage({
          chatId,
          role: "assistant",
          content: formatProjectAttention(result),
          metadata: { route: page.path, runId },
        });
        store.finishChatRun(chatId);
      } catch {
        if (controller.signal.aborted || runToken !== runTokens.get(chatId)) {
          store.removeMessagesByRunId(chatId, runId);
          return;
        }
        store.removeMessagesByRunId(chatId, runId);
        store.addMessage({
          chatId,
          role: "assistant",
          content: t("agent.project-attention.unavailable"),
          metadata: { route: page.path, runId },
        });
        store.finishChatRun(chatId);
      } finally {
        if (store.getAbortController(chatId) === controller) {
          store.setAbortController(chatId, null);
        }
      }
    },
    [formatProjectAttention, t]
  );

  const startChatRun = useCallback(
    async (
      chatId: string,
      currentPage: { path: string; title: string },
      runMode: AgentRunMode = "default",
      starterId?: LandingStarterId,
      skipReferencePreflight = false
    ) => {
      const runId = uuidv4();
      const store = useAgentStore.getState();
      const lastUserMessage = store
        .getMessages(chatId)
        .findLast((message) => message.role === "user");
      const references = normalizeAgentResourceReferences(
        lastUserMessage?.metadata?.resourceReferences,
        lastUserMessage?.metadata?.resourceReference
      );
      const hasResourceReference = references.some(
        (reference) =>
          reference.kind === "project" ||
          reference.kind === "database" ||
          reference.kind === "issue"
      );
      const effectiveRunMode = hasResourceReference ? "read-only" : runMode;
      store.updateChatPage(chatId, currentPage);
      store.startChatRun(chatId, currentPage, {
        runId,
        runMode: effectiveRunMode,
      });
      if (!hasResourceReference && starterId === "workspace-recent") {
        await runRecentActivityStarter(
          chatId,
          currentPage,
          runId,
          "projects/-"
        );
        return;
      }
      if (!hasResourceReference && starterId === "project-open-recent") {
        const projectName = store.getChat(chatId)?.context;
        if (projectName?.startsWith(projectNamePrefix)) {
          await runRecentActivityStarter(
            chatId,
            currentPage,
            runId,
            projectName
          );
          return;
        }
      }
      if (!hasResourceReference && starterId === "workspace-attention") {
        await runWorkspaceAttentionStarter(chatId, currentPage, runId);
        return;
      }
      if (!hasResourceReference && starterId === "project-open-blockers") {
        const projectName = store.getChat(chatId)?.context;
        if (projectName?.startsWith(projectNamePrefix)) {
          await runProjectAttentionStarter(chatId, currentPage, runId);
          return;
        }
      }
      await runChat(
        chatId,
        currentPage,
        runId,
        effectiveRunMode,
        skipReferencePreflight
      );
    },
    [
      runChat,
      runProjectAttentionStarter,
      runRecentActivityStarter,
      runWorkspaceAttentionStarter,
    ]
  );

  useEffect(() => {
    if (!currentChatId || queuedUserMessage?.chatId !== currentChatId) return;
    const store = useAgentStore.getState();
    const message = store.takeQueuedUserMessage(currentChatId);
    if (!message) return;

    void (async () => {
      const restoreQueuedText = () => {
        const draft = store.getDraft(message.chatId);
        if (!draft.text && !draft.project && !draft.resource) {
          store.setDraft(message.chatId, {
            text: message.text,
            resource: message.resourceReference,
          });
        }
      };
      let isBlocked: boolean;
      try {
        isBlocked = await blockWhenAIIsNotConfigured(message.chatId);
      } catch {
        if (store.queuedUserMessage?.id === message.id) {
          restoreQueuedText();
          store.finishQueuedUserMessage(message.chatId);
        }
        return;
      }
      if (isBlocked) {
        restoreQueuedText();
        store.finishQueuedUserMessage(message.chatId);
        return;
      }
      const committed = store.commitQueuedUserMessage(message.chatId);
      if (!committed) return;
      store.clearError(message.chatId);
      await startChatRun(
        message.chatId,
        message.page,
        message.runMode ?? "default",
        message.starterId
      );
    })();
  }, [
    blockWhenAIIsNotConfigured,
    currentChatId,
    queuedUserMessage,
    startChatRun,
  ]);

  const send = useCallback(async () => {
    const store = useAgentStore.getState();
    const chat = selectCurrentChat(store);
    if (store.isChatRunning(chat?.id)) return;

    const currentPage = getCurrentPageSnapshot();
    const sourceThread = store.ensureCurrentChat(
      currentPage,
      getCurrentChatContext(),
      workspace?.name
    );
    const sourceChatId = sourceThread.id;
    const submittedDraft = store.getDraft(sourceChatId);
    const pendingAskBeforeConfig = store.getPendingAsk(sourceChatId);
    if (await blockWhenAIIsNotConfigured(sourceChatId)) return;
    const currentStore = useAgentStore.getState();
    const pendingAsk = currentStore.getPendingAsk(sourceChatId);
    if (pendingAsk?.toolCallId !== pendingAskBeforeConfig?.toolCallId) return;
    if (
      pendingAsk &&
      currentStore.getChat(sourceChatId)?.status !== "awaiting_user"
    ) {
      return;
    }
    currentStore.clearError(sourceChatId);

    if (pendingAsk) {
      const answer = input.trim();
      if (!answer) return;
      let answered;
      if (pendingAsk.kind === "choose") {
        answered = currentStore.answerPendingAsk(
          sourceChatId,
          { kind: "choose", answer, value: answer },
          { route: currentPage.path }
        );
      } else if (pendingAsk.kind === "input") {
        answered = currentStore.answerPendingAsk(
          sourceChatId,
          { kind: "input", answer },
          { route: currentPage.path }
        );
      } else {
        return;
      }
      if (!answered) return;
      setAskInput("");
      await startChatRun(
        sourceChatId,
        currentPage,
        currentStore.getChat(sourceChatId)?.runMode ?? "default"
      );
      return;
    }

    const text = submittedDraft.text.trim();
    if (!text) return;
    const mentions = getDraftResourceMentions(submittedDraft);
    if (
      mentions.length > 0 &&
      hasUnboundResourceMention(submittedDraft.text, mentions)
    )
      return;
    const references = normalizeAgentResourceReferences(
      mentions.map(({ resource }) => resource)
    );
    const stagedReference = references[0];
    const thread =
      stagedReference && !sourceThread.context
        ? store.selectOrCreateChatForContext(
            stagedReference.project,
            currentPage,
            workspace?.name
          )
        : sourceThread;
    if (!thread) return;
    const chatId = thread.id;
    store.addMessage({
      chatId,
      role: "user",
      content: text,
      metadata: {
        route: currentPage.path,
        ...(stagedReference ? { resourceReference: stagedReference } : {}),
        ...(references.length > 1 ? { resourceReferences: references } : {}),
      },
    });
    if (store.getDraft(sourceChatId) === submittedDraft) {
      store.clearDraft(sourceChatId);
    }
    setResourcePickerOpen(false);
    mentionRangeRef.current = null;
    const starterId = getLandingStarterIdForText({
      context: thread.context,
      text,
      translate: t,
    });
    await startChatRun(chatId, currentPage, "default", starterId);
  }, [blockWhenAIIsNotConfigured, input, startChatRun, t, workspace?.name]);

  const retryLastTurn = useCallback(async () => {
    const store = useAgentStore.getState();
    const chat = selectCurrentChat(store);
    if (store.isChatRunning(chat?.id) || !chat?.interrupted) return;

    const chatId = chat.id;
    const currentPage = getCurrentPageSnapshot();
    if (await blockWhenAIIsNotConfigured(chatId)) return;
    store.removeMessagesByRunId(chatId, chat.runId);
    store.clearError(chatId);
    await startChatRun(chatId, currentPage, chat.runMode ?? "default");
  }, [blockWhenAIIsNotConfigured, startChatRun]);

  useEffect(() => {
    if (!retryActionRef) return;
    const retry = () => void retryLastTurn();
    retryActionRef.current = retry;
    return () => {
      if (retryActionRef.current === retry) retryActionRef.current = null;
    };
  }, [retryActionRef, retryLastTurn]);

  const cancelPendingRequest = useCallback(() => {
    const store = useAgentStore.getState();
    const chat = selectCurrentChat(store);
    if (!store.cancelPendingAsk(chat?.id)) return;
    setAskInput("");
  }, []);

  const submitConfirmation = useCallback(
    async (confirmed: boolean) => {
      const store = useAgentStore.getState();
      const pendingAsk = selectCurrentPendingAsk(store);
      const chat = selectCurrentChat(store);
      if (
        !pendingAsk ||
        pendingAsk.kind !== "confirm" ||
        confirmationInFlightRef.current ||
        store.isChatRunning(chat?.id)
      ) {
        return;
      }

      if (pendingAsk.apiPermission && chat) {
        confirmationInFlightRef.current = true;
        let approvalController: AbortController | undefined;
        try {
          const chatId = chat.id;
          const currentPage = getCurrentPageSnapshot();
          const permissionArgs = {
            actionId: pendingAsk.apiPermission.actionId,
            chatId,
            toolCallId: pendingAsk.toolCallId,
            userName: useAppStore.getState().currentUser?.name ?? "",
          };
          const requestAvailable = Boolean(
            getPendingApiCallPreview(permissionArgs)
          );
          if (requestAvailable && (await blockWhenAIIsNotConfigured(chatId))) {
            return;
          }

          const currentStore = useAgentStore.getState();
          if (
            currentStore.getPendingAsk(chatId)?.apiPermission?.actionId !==
              pendingAsk.apiPermission.actionId ||
            currentStore.isChatRunning(chatId)
          ) {
            return;
          }

          const runMode = currentStore.getChat(chatId)?.runMode ?? "default";
          currentStore.setChatStatus(chatId, "running");
          approvalController = new AbortController();
          currentStore.setAbortController(chatId, approvalController);
          const result = await resolvePendingApiCall({
            ...permissionArgs,
            approved: confirmed && requestAvailable,
            signal: approvalController.signal,
          });
          const currentChat = currentStore.getChat(chatId);
          if (
            approvalController.signal.aborted ||
            !currentChat ||
            currentChat.status !== "running" ||
            currentChat.runId !== chat.runId
          ) {
            return;
          }
          currentStore.addMessage({
            chatId,
            role: "tool",
            toolCallId: pendingAsk.toolCallId,
            content: result,
            metadata: {
              route: currentPage.path,
              runId: chat.runId ?? undefined,
            },
          });
          if (
            !requestAvailable ||
            result === '{"error":"Permission request expired."}' ||
            result ===
              '{"error":"Permission request not found or already used."}'
          ) {
            currentStore.addMessage({
              chatId,
              role: "assistant",
              content: t("agent.api-permission-expired"),
              metadata: {
                route: currentPage.path,
                runId: chat.runId ?? undefined,
              },
            });
            currentStore.finishChatRun(chatId);
            return;
          }
          // An approved write can make the selected resource unavailable.
          // Resume from its recorded result; the executor still checks further access.
          await startChatRun(chatId, currentPage, runMode, undefined, true);
        } finally {
          if (
            approvalController &&
            store.getAbortController(chat.id) === approvalController
          ) {
            store.setAbortController(chat.id, null);
          }
          confirmationInFlightRef.current = false;
        }
        return;
      }

      const currentPage = getCurrentPageSnapshot();
      const thread = store.ensureCurrentChat(currentPage);
      const chatId = thread.id;
      const answer = confirmed ? confirmLabel : cancelLabel;

      if (await blockWhenAIIsNotConfigured(chatId)) return;
      const currentStore = useAgentStore.getState();
      if (
        currentStore.getChat(chatId)?.status !== "awaiting_user" ||
        currentStore.getPendingAsk(chatId)?.toolCallId !== pendingAsk.toolCallId
      ) {
        return;
      }
      currentStore.clearError(chatId);
      const answered = currentStore.answerPendingAsk(
        chatId,
        { kind: "confirm", answer, confirmed },
        { route: currentPage.path }
      );
      if (!answered) return;
      await startChatRun(
        chatId,
        currentPage,
        currentStore.getChat(chatId)?.runMode ?? "default"
      );
    },
    [blockWhenAIIsNotConfigured, confirmLabel, cancelLabel, startChatRun, t]
  );

  const submitChoice = useCallback(
    async (option: AgentAskUserOption) => {
      const store = useAgentStore.getState();
      const pendingAsk = selectCurrentPendingAsk(store);
      const chat = selectCurrentChat(store);
      if (
        !pendingAsk ||
        pendingAsk.kind !== "choose" ||
        store.isChatRunning(chat?.id)
      ) {
        return;
      }

      const currentPage = getCurrentPageSnapshot();
      const thread = store.ensureCurrentChat(currentPage);
      const chatId = thread.id;

      if (await blockWhenAIIsNotConfigured(chatId)) return;
      const currentStore = useAgentStore.getState();
      if (
        currentStore.getChat(chatId)?.status !== "awaiting_user" ||
        currentStore.getPendingAsk(chatId)?.toolCallId !== pendingAsk.toolCallId
      ) {
        return;
      }
      currentStore.clearError(chatId);
      const answered = currentStore.answerPendingAsk(
        chatId,
        { kind: "choose", answer: option.label, value: option.value },
        { route: currentPage.path }
      );
      if (!answered) return;
      await startChatRun(
        chatId,
        currentPage,
        currentStore.getChat(chatId)?.runMode ?? "default"
      );
    },
    [blockWhenAIIsNotConfigured, startChatRun]
  );

  const handleTextareaKeydown = useCallback(
    (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (event.nativeEvent.isComposing) return;
      if (resourcePickerOpen && canMentionResource) {
        const range = getMentionTriggerRange(
          input,
          event.currentTarget.selectionStart
        );
        if (!range) {
          setResourcePickerOpen(false);
          mentionRangeRef.current = null;
        } else if (event.key === "Escape") {
          event.preventDefault();
          handleResourcePickerOpenChange(false);
        } else if (
          event.key === "Enter" &&
          !event.shiftKey &&
          !event.altKey &&
          !event.ctrlKey &&
          !event.metaKey
        ) {
          event.preventDefault();
        }
        if (range) return;
      }
      if (
        event.key === "Enter" &&
        !event.shiftKey &&
        !event.altKey &&
        !event.ctrlKey &&
        !event.metaKey
      ) {
        event.preventDefault();
        void send();
      }
    },
    [
      canMentionResource,
      handleResourcePickerOpenChange,
      input,
      resourcePickerOpen,
      send,
    ]
  );

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  const issueDecisionPermissionText = (impact: IssueDecisionImpact["kind"]) => {
    switch (impact) {
      case "reject":
        return t("agent.permission.issue-rejection");
      case "final-approval-with-rollout":
        return t("agent.permission.issue-approval-rollout");
      case "final-approval-with-access-grant":
        return t("agent.permission.issue-approval-access-grant");
      case "final-approval-with-role-grant":
        return t("agent.permission.issue-approval-role-grant");
      case "final-approval":
        return t("agent.permission.issue-approval-final");
      default:
        return t("agent.permission.issue-approval");
    }
  };

  return (
    <div ref={composerRef} className="border-t p-3">
      {/* Pending ask banner */}
      {currentPendingAsk && (
        <div className="mb-3 rounded-xs bg-warning/10 px-3 py-2 text-xs text-warning">
          {apiPermission ? (
            <>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="font-medium">
                  {apiPermissionPreview
                    ? t("agent.api-permission-title")
                    : t("agent.api-permission-expired-title")}
                </div>
                {apiPermissionPreview && (
                  <span role="timer" className="tabular-nums">
                    {t("agent.api-permission-expires-in", {
                      time: formatPermissionTime(
                        apiPermissionPreview.expiresAt
                      ),
                    })}
                  </span>
                )}
              </div>
              {apiPermissionPreview ? (
                <>
                  <div className="mt-1">
                    {t("agent.api-permission-request")}
                  </div>
                  <div className="mt-2 font-mono break-all">
                    {apiPermissionPreview.operationId}
                  </div>
                  {apiPermissionPreview.issueDecision && (
                    <div className="mt-2 border-l-2 border-warning pl-2">
                      <div className="font-medium">
                        {apiPermissionPreview.issueDecision.issueTitle}
                      </div>
                      <div className="font-mono break-all text-control-light">
                        {apiPermissionPreview.issueDecision.issueName}
                      </div>
                      <div className="mt-1">
                        {issueDecisionPermissionText(
                          apiPermissionPreview.issueDecision.impact
                        )}
                      </div>
                    </div>
                  )}
                  <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-all rounded-xs border bg-background p-2 font-mono text-control">
                    {JSON.stringify(
                      JSON.parse(apiPermissionPreview.body),
                      null,
                      2
                    )}
                  </pre>
                </>
              ) : (
                <div className="mt-1">{t("agent.api-permission-expired")}</div>
              )}
            </>
          ) : (
            <>
              <div className="font-medium">{currentPendingAsk.prompt}</div>
              <div className="mt-1">
                {currentPendingAsk.kind === "confirm"
                  ? t("agent.pending-confirm-hint")
                  : currentPendingAsk.kind === "choose"
                    ? t("agent.pending-choose-hint")
                    : t("agent.pending-input-hint")}
              </div>
              {canCancelPendingRequest && (
                <div className="mt-2 flex justify-end">
                  <Button
                    appearance="secondary"
                    size="sm"
                    className="rounded-xs border text-control hover:bg-control-bg"
                    onClick={cancelPendingRequest}
                  >
                    {t("agent.cancel-request")}
                  </Button>
                </div>
              )}
            </>
          )}
        </div>
      )}

      <div
        className="mb-1 flex items-center justify-between gap-2 text-xs text-control-light"
        data-agent-input-footer
      >
        <AgentContextPicker
          context={context}
          contextTitle={contextTitle}
          workspace={workspace}
          disabled={hasRunningChat || Boolean(currentPendingAsk)}
          onSelect={selectContext}
        />
        <span className="shrink-0">{currentChatTokenUsageLabel}</span>
      </div>

      {/* Confirm buttons */}
      {isAwaitingConfirm ? (
        <div className="flex flex-wrap gap-x-2 gap-y-2">
          <Button
            appearance="secondary"
            size={apiPermission ? "sm" : "md"}
            className="rounded-xs bg-accent text-accent-text hover:bg-accent-hover disabled:opacity-50"
            disabled={
              isCurrentChatRunning || (apiPermission && !apiPermissionPreview)
            }
            onClick={() => submitConfirmation(true)}
          >
            {apiPermission ? t("agent.api-permission-allow") : confirmLabel}
          </Button>
          <Button
            appearance="secondary"
            size={apiPermission ? "sm" : "md"}
            className="rounded-xs border text-control hover:bg-control-bg disabled:opacity-50"
            disabled={isCurrentChatRunning}
            onClick={() => submitConfirmation(false)}
          >
            {apiPermission
              ? apiPermissionPreview
                ? t("agent.api-permission-deny")
                : t("agent.api-permission-dismiss")
              : cancelLabel}
          </Button>
        </div>
      ) : isAwaitingChoose ? (
        /* Choose options */
        <div className="flex flex-col gap-y-2">
          {chooseOptions.map((option) => (
            <Button
              appearance="secondary"
              size="md"
              key={option.value}
              className="h-auto w-full flex-col items-start justify-start rounded-xs border px-3 py-2 text-left text-sm whitespace-normal hover:bg-control-bg disabled:opacity-50"
              disabled={isCurrentChatRunning}
              onClick={() => submitChoice(option)}
            >
              <div className="w-full min-w-0 break-words font-medium text-main">
                {option.label}
              </div>
              {option.description && (
                <div className="mt-1 w-full min-w-0 break-words text-xs text-control-light">
                  {option.description}
                </div>
              )}
            </Button>
          ))}
        </div>
      ) : (
        <div className="relative flex items-end gap-x-2" data-agent-input-row>
          <div className="relative w-full">
            {!input && (
              <div
                className="pointer-events-none absolute inset-x-3 inset-y-0 z-10 flex items-center truncate text-sm leading-5 text-control-light"
                data-agent-input-placeholder
              >
                {inputPlaceholder} · {t("agent.input-keyboard-hint")}
              </div>
            )}
            <AgentMentionTextarea
              size="md"
              textareaRef={textareaRef}
              value={input}
              mentions={draftMentions.map(({ resource, range }) => ({
                range,
                label: resource.label,
              }))}
              onDeleteMention={({ text, caret }) => {
                const store = useAgentStore.getState();
                if (!store.currentChatId) return;
                const draft = store.getDraft(store.currentChatId);
                selectedMentionCaretRef.current = caret;
                mentionRangeRef.current = null;
                setResourcePickerOpen(false);
                store.setDraft(store.currentChatId, {
                  text,
                  mentions: updateDraftResourceMentions(
                    draft.text,
                    text,
                    getDraftResourceMentions(draft)
                  ),
                });
              }}
              rows={1}
              className="block h-9 min-h-9 w-full resize-none overflow-y-hidden rounded-xs border px-3 py-1.5 text-sm leading-5 outline-none focus:ring-1 focus:ring-accent disabled:opacity-50"
              aria-label={inputPlaceholder}
              disabled={isCurrentChatRunning || isAIConfigurationBlocked}
              onChange={(event) => {
                setInput(event.target.value);
                if (!canMentionResource) {
                  mentionRangeRef.current = null;
                  setResourcePickerOpen(false);
                  return;
                }
                const range = getMentionTriggerRange(
                  event.target.value,
                  event.target.selectionStart
                );
                mentionRangeRef.current = range;
                setResourcePickerOpen(Boolean(range));
              }}
              onKeyDown={handleTextareaKeydown}
            />
            {canMentionResource && workspace?.name ? (
              <AgentResourcePicker
                open={resourcePickerOpen}
                onOpenChange={handleResourcePickerOpenChange}
                anchorRef={textareaRef}
                workspaceName={workspace.name}
                projectName={projectContext}
                selectedProjectName={selectedProjectMention?.resource.name}
                hasSelectedReference={
                  draftMentions.length > 1 ||
                  (draftMentions.length === 1 && !selectedProjectMention)
                }
                onSelectProject={selectMentionProject}
                onSelectDatabase={selectMentionDatabase}
                onSelectIssue={selectMentionIssue}
              />
            ) : null}
          </div>

          {loading ? (
            <Button
              variant="destructive"
              size="md"
              className="shrink-0 self-end whitespace-nowrap"
              onClick={() =>
                useAgentStore.getState().cancel(currentChat?.id, "user")
              }
            >
              {t("agent.stop")}
            </Button>
          ) : (
            <Button
              size="md"
              className="shrink-0 self-end whitespace-nowrap"
              data-agent-send
              disabled={isSendDisabled}
              onClick={() => void send()}
            >
              {sendLabel}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
