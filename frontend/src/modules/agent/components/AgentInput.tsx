import {
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
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { runAgentLoop } from "../logic/agentLoop";
import {
  getAgentAIConfigurationEnabled,
  isAgentAIConfigurationError,
} from "../logic/aiConfiguration";
import { getCurrentChatContext } from "../logic/chatContext";
import {
  getLandingStarterIdForText,
  type LandingStarterId,
} from "../logic/landingStarters";
import { buildOutboundHistory } from "../logic/outboundHistory";
import { buildSystemPrompt } from "../logic/prompt";
import {
  getMentionTriggerRange,
  hasAdditionalResourceMention,
  insertResourceMention,
  updateResourceMentionRange,
} from "../logic/resourceMention";
import { createToolExecutor, getToolDefinitions } from "../logic/tools";
import { extractDatabaseOverviewEvidence } from "../logic/tools/databaseOverview";
import { extractIssueOverviewEvidence } from "../logic/tools/issueOverview";
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

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

const getCurrentPageSnapshot = () => ({
  path: router.currentRoute.value.fullPath,
  title: document.title,
});

const projectReferenceFromDraft = (draft: {
  project?: { name: string; title: string };
  resource?: AgentResourceReference;
}): AgentResourceReference | undefined =>
  draft.resource?.kind === "project"
    ? draft.resource
    : draft.project
      ? {
          kind: "project",
          name: draft.project.name,
          project: draft.project.name,
          label: draft.project.title,
        }
      : undefined;

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

export function AgentInput() {
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
  const [resourcePickerOpen, setResourcePickerOpen] = useState(false);
  const mentionRangeRef = useRef<{ start: number; end: number } | null>(null);
  const selectedMentionCaretRef = useRef<number | null>(null);
  const previousChatIdRef = useRef<string | null>(null);

  const composerRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Derived values
  const isInterrupted = Boolean(currentChat?.interrupted);
  const isCurrentChatRunning = currentChat?.status === "running";
  const chooseOptions =
    currentPendingAsk?.kind === "choose"
      ? (currentPendingAsk.options ?? [])
      : [];
  const isAwaitingConfirm = currentPendingAsk?.kind === "confirm";
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
      const resource = draft.resource ?? projectReferenceFromDraft(draft);
      const mention = resource
        ? updateResourceMentionRange(
            draft.text,
            value,
            draft.mention ?? null,
            resource.label
          )
        : null;
      store.setDraft(chatId, {
        text: value,
        ...(mention && resource ? { mention, resource } : {}),
      });
    },
    [currentPendingAsk, workspace?.name]
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
  const hasAdditionalMention = Boolean(
    !currentPendingAsk &&
      currentDraft?.mention &&
      (currentDraft.resource || currentDraft.project) &&
      hasAdditionalResourceMention(input, currentDraft.mention)
  );
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
  const canMentionProject = Boolean(workspace?.name && !currentPendingAsk);

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

  const selectMentionProject = useCallback(
    (project: Project) => {
      const store = useAgentStore.getState();
      const chatId =
        store.currentChatId ??
        store.ensureCurrentChat(
          getCurrentPageSnapshot(),
          getCurrentChatContext(),
          workspace?.name
        ).id;
      const draft = store.getDraft(chatId);
      if (draft.resource || draft.project) return;
      const range = mentionRangeRef.current;
      const label = project.title || project.name;
      const insertion = insertResourceMention(draft.text, range, label);
      if (!insertion) return;
      selectedMentionCaretRef.current = insertion.caret;
      store.setDraft(chatId, {
        text: insertion.text,
        mention: insertion.mention,
        resource: {
          kind: "project",
          name: project.name,
          project: project.name,
          label,
        },
      });
      mentionRangeRef.current = null;
    },
    [workspace?.name]
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
      runMode: AgentRunMode
    ) => {
      const store = useAgentStore.getState();
      const lastUserMessage = store
        .getMessages(chatId)
        .findLast((message) => message.role === "user");
      const referencedProject =
        lastUserMessage?.metadata?.resourceReference?.kind === "project"
          ? lastUserMessage.metadata.resourceReference.name
          : undefined;
      const controller = new AbortController();
      const runToken = (runTokens.get(chatId) ?? 0) + 1;
      runTokens.set(chatId, runToken);
      store.setAbortController(chatId, controller);

      const systemPrompt = buildSystemPrompt({
        ...page,
        context: store.getChat(chatId)?.context,
        referencedProject,
        runMode,
      });
      const tools = getToolDefinitions({ runMode, referencedProject });
      const executor = createToolExecutor(router, {
        chatId,
        context: store.getChat(chatId)?.context,
        referencedProject,
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
        if (referencedProject) {
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
      starterId?: LandingStarterId
    ) => {
      const runId = uuidv4();
      const store = useAgentStore.getState();
      const lastUserMessage = store
        .getMessages(chatId)
        .findLast((message) => message.role === "user");
      const hasProjectReference =
        lastUserMessage?.metadata?.resourceReference?.kind === "project";
      const effectiveRunMode = hasProjectReference ? "read-only" : runMode;
      store.updateChatPage(chatId, currentPage);
      store.startChatRun(chatId, currentPage, {
        runId,
        runMode: effectiveRunMode,
      });
      if (!hasProjectReference && starterId === "workspace-recent") {
        await runRecentActivityStarter(
          chatId,
          currentPage,
          runId,
          "projects/-"
        );
        return;
      }
      if (!hasProjectReference && starterId === "project-open-recent") {
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
      if (!hasProjectReference && starterId === "workspace-attention") {
        await runWorkspaceAttentionStarter(chatId, currentPage, runId);
        return;
      }
      if (!hasProjectReference && starterId === "project-open-blockers") {
        const projectName = store.getChat(chatId)?.context;
        if (projectName?.startsWith(projectNamePrefix)) {
          await runProjectAttentionStarter(chatId, currentPage, runId);
          return;
        }
      }
      await runChat(chatId, currentPage, runId, effectiveRunMode);
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
    if (await blockWhenAIIsNotConfigured(sourceChatId)) return;
    store.clearError(sourceChatId);

    const pendingAsk = selectCurrentPendingAsk(store);
    if (pendingAsk) {
      const answer = input.trim();
      if (!answer) return;
      setAskInput("");

      if (pendingAsk.kind === "choose") {
        store.answerPendingAsk(
          sourceChatId,
          { kind: "choose", answer, value: answer },
          { route: currentPage.path }
        );
      } else if (pendingAsk.kind === "input") {
        store.answerPendingAsk(
          sourceChatId,
          { kind: "input", answer },
          { route: currentPage.path }
        );
      } else {
        return;
      }
      await startChatRun(
        sourceChatId,
        currentPage,
        store.getChat(sourceChatId)?.runMode ?? "default"
      );
      return;
    }

    const text = submittedDraft.text.trim();
    if (!text) return;
    const stagedReference = projectReferenceFromDraft(submittedDraft);
    if (
      stagedReference &&
      submittedDraft.mention &&
      hasAdditionalResourceMention(submittedDraft.text, submittedDraft.mention)
    ) {
      return;
    }
    const thread =
      stagedReference && !sourceThread.context
        ? store.selectOrCreateChatForContext(
            stagedReference.name,
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

  const dismissInterrupted = useCallback(() => {
    const store = useAgentStore.getState();
    const chat = selectCurrentChat(store);
    if (!chat) return;
    store.removeMessagesByRunId(chat.id, chat.runId);
    store.clearError(chat.id);
  }, []);

  const submitConfirmation = useCallback(
    async (confirmed: boolean) => {
      const store = useAgentStore.getState();
      const pendingAsk = selectCurrentPendingAsk(store);
      const chat = selectCurrentChat(store);
      if (
        !pendingAsk ||
        pendingAsk.kind !== "confirm" ||
        store.isChatRunning(chat?.id)
      ) {
        return;
      }

      const currentPage = getCurrentPageSnapshot();
      const thread = store.ensureCurrentChat(currentPage);
      const chatId = thread.id;
      const answer = confirmed ? confirmLabel : cancelLabel;

      if (await blockWhenAIIsNotConfigured(chatId)) return;
      store.clearError(chatId);
      store.answerPendingAsk(
        chatId,
        { kind: "confirm", answer, confirmed },
        { route: currentPage.path }
      );
      await startChatRun(
        chatId,
        currentPage,
        store.getChat(chatId)?.runMode ?? "default"
      );
    },
    [blockWhenAIIsNotConfigured, confirmLabel, cancelLabel, startChatRun]
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
      store.clearError(chatId);
      store.answerPendingAsk(
        chatId,
        { kind: "choose", answer: option.label, value: option.value },
        { route: currentPage.path }
      );
      await startChatRun(
        chatId,
        currentPage,
        store.getChat(chatId)?.runMode ?? "default"
      );
    },
    [blockWhenAIIsNotConfigured, startChatRun]
  );

  const handleTextareaKeydown = useCallback(
    (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (event.nativeEvent.isComposing) return;
      if (resourcePickerOpen && canMentionProject) {
        const range = getMentionTriggerRange(
          input,
          event.currentTarget.selectionStart
        );
        if (!range) {
          setResourcePickerOpen(false);
          mentionRangeRef.current = null;
        } else if (event.key === "Escape") {
          event.preventDefault();
          setResourcePickerOpen(false);
          mentionRangeRef.current = null;
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
    [canMentionProject, input, resourcePickerOpen, send]
  );

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <div ref={composerRef} className="border-t p-3">
      {/* Pending ask banner */}
      {currentPendingAsk && (
        <div className="mb-3 rounded-xs bg-warning/10 px-3 py-2 text-xs text-warning">
          <div className="font-medium">{currentPendingAsk.prompt}</div>
          <div className="mt-1">
            {currentPendingAsk.kind === "confirm"
              ? t("agent.pending-confirm-hint")
              : currentPendingAsk.kind === "choose"
                ? t("agent.pending-choose-hint")
                : t("agent.pending-input-hint")}
          </div>
        </div>
      )}

      {/* Interrupted banner */}
      {isInterrupted && (
        <div className="mb-3 rounded-xs bg-error/10 px-3 py-2 text-xs text-error">
          <div className="font-medium">{t("agent.interrupted")}</div>
          <div className="mt-1">{t("agent.interrupted-retry-hint")}</div>
          <div className="mt-2 flex flex-wrap gap-x-2 gap-y-2">
            <Button
              appearance="secondary"
              size="md"
              className="rounded-xs bg-error px-3 py-2 text-sm text-accent-text hover:bg-error/90 disabled:opacity-50"
              disabled={isCurrentChatRunning}
              onClick={retryLastTurn}
            >
              {t("agent.retry-last-chat-turn")}
            </Button>
            <Button
              appearance="secondary"
              size="md"
              className="rounded-xs border px-3 py-2 text-sm text-control hover:bg-control-bg disabled:opacity-50"
              disabled={isCurrentChatRunning}
              onClick={dismissInterrupted}
            >
              {t("common.dismiss")}
            </Button>
          </div>
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
            size="md"
            className="rounded-xs bg-accent px-3 py-2 text-sm text-accent-text hover:bg-accent-hover disabled:opacity-50"
            disabled={isCurrentChatRunning}
            onClick={() => submitConfirmation(true)}
          >
            {confirmLabel}
          </Button>
          <Button
            appearance="secondary"
            size="md"
            className="rounded-xs border px-3 py-2 text-sm text-control hover:bg-control-bg disabled:opacity-50"
            disabled={isCurrentChatRunning}
            onClick={() => submitConfirmation(false)}
          >
            {cancelLabel}
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
              mention={currentPendingAsk ? null : currentDraft?.mention}
              mentionLabel={
                currentDraft?.resource?.label ?? currentDraft?.project?.title
              }
              onDeleteMention={({ text, caret }) => {
                const store = useAgentStore.getState();
                if (!store.currentChatId) return;
                selectedMentionCaretRef.current = caret;
                mentionRangeRef.current = null;
                setResourcePickerOpen(false);
                store.setDraft(store.currentChatId, { text });
              }}
              rows={1}
              className="block h-9 min-h-9 w-full resize-none overflow-y-hidden rounded-xs border px-3 py-1.5 text-sm leading-5 outline-none focus:ring-1 focus:ring-accent disabled:opacity-50"
              aria-label={inputPlaceholder}
              disabled={isCurrentChatRunning || isAIConfigurationBlocked}
              onChange={(event) => {
                setInput(event.target.value);
                if (!canMentionProject) {
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
            {canMentionProject && workspace?.name ? (
              <AgentResourcePicker
                open={resourcePickerOpen}
                onOpenChange={setResourcePickerOpen}
                anchorRef={textareaRef}
                workspaceName={workspace.name}
                hasSelectedReference={Boolean(
                  currentDraft?.resource || currentDraft?.project
                )}
                onSelectProject={selectMentionProject}
              />
            ) : null}
          </div>

          {loading ? (
            <Button
              variant="destructive"
              size="md"
              className="shrink-0 self-end whitespace-nowrap"
              onClick={() => useAgentStore.getState().cancel(currentChat?.id)}
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
