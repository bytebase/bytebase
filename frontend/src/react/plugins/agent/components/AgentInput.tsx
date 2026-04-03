import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { Button } from "@/react/components/ui/button";
import { router } from "@/router";
import type { DomRefSuggestion } from "../dom";
import { lazyExtractDomRefSuggestions } from "../dom";
import { runAgentLoop } from "../logic/agentLoop";
import { isAgentAIConfigurationError } from "../logic/aiConfiguration";
import { buildOutboundHistory } from "../logic/outboundHistory";
import { buildSystemPrompt } from "../logic/prompt";
import { createToolExecutor, getToolDefinitions } from "../logic/tools";
import type { AgentAskUserOption, Message } from "../logic/types";
import {
  selectCurrentChat,
  selectCurrentChatRequiresAIConfiguration,
  selectCurrentPendingAsk,
  selectLoading,
  useAgentStore,
} from "../store/agent";

// Module-level map — survives across renders, same as Vue version.
const runTokens = new Map<string, number>();

// Module-level request token for dom ref suggestion loading.
let domRefRequestToken = 0;

// ---------------------------------------------------------------------------
// Helper functions (same logic as the Vue version)
// ---------------------------------------------------------------------------

const normalizeSearchText = (value?: string) =>
  value?.toLowerCase().trim() ?? "";

const matchDomRefSuggestion = (suggestion: DomRefSuggestion, query: string) => {
  if (!query) return true;
  const normalizedQuery = normalizeSearchText(query);
  return [
    suggestion.ref,
    suggestion.tag,
    suggestion.role,
    suggestion.label,
    suggestion.value,
  ].some((value) => normalizeSearchText(value).includes(normalizedQuery));
};

const getDomRefQuery = (
  text: string,
  start: number,
  end: number
): { query: string; start: number; end: number } | null => {
  if (start !== end) return null;
  const prefix = text.slice(0, start);
  const matches = prefix.match(/(^|\s)@([^\s@]*)$/);
  if (!matches) return null;
  return {
    query: matches[2] ?? "",
    start: start - matches[2].length - 1,
    end,
  };
};

const formatDomRefSuggestionMeta = (suggestion: DomRefSuggestion) =>
  [suggestion.tag.toLowerCase(), suggestion.role]
    .filter((value): value is string => Boolean(value))
    .join(" · ");

const getCurrentPageSnapshot = () => ({
  path: router.currentRoute.value.fullPath,
  title: document.title,
});

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface DomRefMentionOption {
  label: string;
  value: string;
  suggestion: DomRefSuggestion;
}

export function AgentInput() {
  const { t } = useTranslation();

  // Zustand selectors
  const currentChat = useAgentStore(selectCurrentChat);
  const currentPendingAsk = useAgentStore(selectCurrentPendingAsk);
  const loading = useAgentStore(selectLoading);
  const isAIConfigurationBlocked = useAgentStore(
    selectCurrentChatRequiresAIConfiguration
  );
  const currentChatId = useAgentStore((s) => s.currentChatId);

  // Local state
  const [input, setInput] = useState("");
  const [selectionStart, setSelectionStart] = useState(0);
  const [selectionEnd, setSelectionEnd] = useState(0);
  const [domRefSuggestions, setDomRefSuggestions] = useState<
    DomRefSuggestion[]
  >([]);
  const [isMentionOpen, setIsMentionOpen] = useState(false);
  const [highlightIndex, setHighlightIndex] = useState(0);

  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const mentionListRef = useRef<HTMLDivElement>(null);

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
  const sendLabel = currentPendingAsk ? t("agent.reply") : t("agent.send");
  const inputPlaceholder =
    currentPendingAsk?.kind === "input"
      ? currentPendingAsk.prompt
      : currentPendingAsk?.kind === "choose"
        ? currentPendingAsk.prompt
        : t("agent.input-placeholder");
  const confirmLabel = currentPendingAsk?.confirmLabel ?? t("agent.confirm");
  const cancelLabel = currentPendingAsk?.cancelLabel ?? t("agent.cancel");
  const isSendDisabled =
    isCurrentChatRunning ||
    isAwaitingConfirm ||
    isAwaitingChoose ||
    isAIConfigurationBlocked ||
    !input.trim();

  // @-mention autocomplete
  const activeDomRefQuery = useMemo(
    () => getDomRefQuery(input, selectionStart, selectionEnd),
    [input, selectionStart, selectionEnd]
  );

  const filteredSuggestions = useMemo(() => {
    const query = activeDomRefQuery?.query ?? "";
    return domRefSuggestions.filter((s) => matchDomRefSuggestion(s, query));
  }, [domRefSuggestions, activeDomRefQuery]);

  const mentionOptions = useMemo<DomRefMentionOption[]>(
    () =>
      filteredSuggestions.map((suggestion) => ({
        label: `[${suggestion.ref}] ${suggestion.label}`,
        value: `[${suggestion.ref}]`,
        suggestion,
      })),
    [filteredSuggestions]
  );

  // Show/hide mention popover
  const showMention = isMentionOpen && mentionOptions.length > 0;

  // Reset highlight when options change
  useEffect(() => {
    setHighlightIndex(0);
  }, [mentionOptions.length]);

  // Close mention menu when chat starts running
  useEffect(() => {
    if (isCurrentChatRunning) {
      setIsMentionOpen(false);
    }
  }, [isCurrentChatRunning]);

  // Reset input when chat/pendingAsk changes
  useEffect(() => {
    if (
      currentPendingAsk?.kind === "input" ||
      currentPendingAsk?.kind === "choose"
    ) {
      setInput(currentPendingAsk.defaultValue ?? "");
      return;
    }
    setInput("");
  }, [
    currentChatId,
    currentPendingAsk?.toolCallId,
    currentPendingAsk?.kind,
    currentPendingAsk?.defaultValue,
  ]);

  // Load dom ref suggestions when @ query is active
  useEffect(() => {
    if (!activeDomRefQuery) {
      setDomRefSuggestions([]);
      setIsMentionOpen(false);
      return;
    }
    setIsMentionOpen(true);

    // Only load once (when suggestions are empty)
    if (domRefSuggestions.length > 0) return;

    const requestToken = ++domRefRequestToken;
    void lazyExtractDomRefSuggestions().then((suggestions) => {
      if (requestToken !== domRefRequestToken) return;
      setDomRefSuggestions(suggestions);
    });
  }, [input, selectionStart, selectionEnd]);

  // Auto-resize textarea
  const autoResize = useCallback(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = "auto";
    const maxRows = 6;
    const lineHeight = 20;
    const maxHeight = lineHeight * maxRows;
    el.style.height = `${Math.min(el.scrollHeight, maxHeight)}px`;
  }, []);

  useEffect(() => {
    autoResize();
  }, [input, autoResize]);

  // Selection tracking
  const updateSelection = useCallback(() => {
    const el = textareaRef.current;
    if (!el) return;
    setSelectionStart(el.selectionStart ?? 0);
    setSelectionEnd(el.selectionEnd ?? 0);
  }, []);

  // Select a mention option
  const selectMention = useCallback(
    (option: DomRefMentionOption) => {
      if (!activeDomRefQuery) return;
      const before = input.slice(0, activeDomRefQuery.start);
      const after = input.slice(activeDomRefQuery.end);
      const newValue = `${before}${option.value} ${after}`;
      setInput(newValue);
      setIsMentionOpen(false);
      setDomRefSuggestions([]);

      // Set cursor after the inserted mention
      const cursorPos = before.length + option.value.length + 1;
      requestAnimationFrame(() => {
        const el = textareaRef.current;
        if (el) {
          el.focus();
          el.setSelectionRange(cursorPos, cursorPos);
          setSelectionStart(cursorPos);
          setSelectionEnd(cursorPos);
        }
      });
    },
    [activeDomRefQuery, input]
  );

  // Scroll highlighted item into view
  const scrollHighlightedIntoView = useCallback((index: number) => {
    requestAnimationFrame(() => {
      const container = mentionListRef.current;
      if (!container) return;
      const items = container.querySelectorAll("[data-mention-option]");
      items[index]?.scrollIntoView({ block: "nearest" });
    });
  }, []);

  // ---------------------------------------------------------------------------
  // Agent loop execution (same logic as Vue version)
  // ---------------------------------------------------------------------------

  const buildChatHistory = useCallback(
    (chatId: string, systemPrompt: string): Message[] => {
      return buildOutboundHistory(
        systemPrompt,
        useAgentStore.getState().getMessages(chatId)
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
      runId: string
    ) => {
      const store = useAgentStore.getState();
      const controller = new AbortController();
      const runToken = (runTokens.get(chatId) ?? 0) + 1;
      runTokens.set(chatId, runToken);
      store.setAbortController(chatId, controller);

      const systemPrompt = buildSystemPrompt(page);
      const tools = getToolDefinitions();
      const executor = createToolExecutor(router, {
        chatId,
        onNavigate: () => {
          useAgentStore
            .getState()
            .updateChatPage(chatId, getCurrentPageSnapshot());
        },
      });

      try {
        const outcome = await runAgentLoop(
          buildChatHistory(chatId, systemPrompt),
          tools,
          executor,
          {
            onAssistantMessage: (message) => {
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
                metadata: { route: page.path, runId },
              });
            },
          },
          controller.signal
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
    [buildChatHistory, handleOutcome]
  );

  const startChatRun = useCallback(
    async (chatId: string, currentPage: { path: string; title: string }) => {
      const runId = uuidv4();
      const store = useAgentStore.getState();
      store.updateChatPage(chatId, currentPage);
      store.startChatRun(chatId, currentPage, { runId });
      await runChat(chatId, currentPage, runId);
    },
    [runChat]
  );

  const send = useCallback(async () => {
    const store = useAgentStore.getState();
    const chat = selectCurrentChat(store);
    if (store.isChatRunning(chat?.id)) return;

    const currentPage = getCurrentPageSnapshot();
    const thread = store.ensureCurrentChat(currentPage);
    const chatId = thread.id;
    store.clearError(chatId);

    const pendingAsk = selectCurrentPendingAsk(store);
    if (pendingAsk) {
      const answer = input.trim();
      if (!answer) return;
      setInput("");

      if (pendingAsk.kind === "choose") {
        store.answerPendingAsk(
          chatId,
          { kind: "choose", answer, value: answer },
          { route: currentPage.path }
        );
      } else if (pendingAsk.kind === "input") {
        store.answerPendingAsk(
          chatId,
          { kind: "input", answer },
          { route: currentPage.path }
        );
      } else {
        return;
      }
      await startChatRun(chatId, currentPage);
      return;
    }

    const text = input.trim();
    if (!text) return;
    setInput("");
    store.addMessage({
      chatId,
      role: "user",
      content: text,
      metadata: { route: currentPage.path },
    });
    await startChatRun(chatId, currentPage);
  }, [input, startChatRun]);

  const retryLastTurn = useCallback(async () => {
    const store = useAgentStore.getState();
    const chat = selectCurrentChat(store);
    if (store.isChatRunning(chat?.id) || !chat?.interrupted) return;

    const chatId = chat.id;
    const currentPage = getCurrentPageSnapshot();
    store.removeMessagesByRunId(chatId, chat.runId);
    store.clearError(chatId);
    await startChatRun(chatId, currentPage);
  }, [startChatRun]);

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

      store.clearError(chatId);
      store.answerPendingAsk(
        chatId,
        { kind: "confirm", answer, confirmed },
        { route: currentPage.path }
      );
      await startChatRun(chatId, currentPage);
    },
    [confirmLabel, cancelLabel, startChatRun]
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

      store.clearError(chatId);
      store.answerPendingAsk(
        chatId,
        { kind: "choose", answer: option.label, value: option.value },
        { route: currentPage.path }
      );
      await startChatRun(chatId, currentPage);
    },
    [startChatRun]
  );

  // Textarea keydown handler
  const handleTextareaKeydown = useCallback(
    (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (showMention) {
        if (event.key === "ArrowDown") {
          event.preventDefault();
          const next = Math.min(highlightIndex + 1, mentionOptions.length - 1);
          setHighlightIndex(next);
          scrollHighlightedIntoView(next);
          return;
        }
        if (event.key === "ArrowUp") {
          event.preventDefault();
          const prev = Math.max(highlightIndex - 1, 0);
          setHighlightIndex(prev);
          scrollHighlightedIntoView(prev);
          return;
        }
        if (event.key === "Enter") {
          event.preventDefault();
          if (mentionOptions[highlightIndex]) {
            selectMention(mentionOptions[highlightIndex]);
          }
          return;
        }
        if (event.key === "Escape") {
          event.preventDefault();
          setIsMentionOpen(false);
          return;
        }
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
      showMention,
      highlightIndex,
      mentionOptions,
      selectMention,
      scrollHighlightedIntoView,
      send,
    ]
  );

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <div className="border-t p-3">
      {/* Pending ask banner */}
      {currentPendingAsk && (
        <div className="mb-3 rounded-md bg-amber-50 px-3 py-2 text-xs text-amber-700">
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
        <div className="mb-3 rounded-md bg-red-50 px-3 py-2 text-xs text-red-700">
          <div className="font-medium">{t("agent.interrupted")}</div>
          <div className="mt-1">{t("agent.interrupted-retry-hint")}</div>
          <div className="mt-2 flex flex-wrap gap-x-2 gap-y-2">
            <button
              className="rounded-md bg-red-500 px-3 py-2 text-sm text-white hover:bg-red-600 disabled:opacity-50"
              disabled={isCurrentChatRunning}
              onClick={retryLastTurn}
            >
              {t("agent.retry-last-chat-turn")}
            </button>
            <button
              className="rounded-md border px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 disabled:opacity-50"
              disabled={isCurrentChatRunning}
              onClick={dismissInterrupted}
            >
              {t("common.dismiss")}
            </button>
          </div>
        </div>
      )}

      {/* Confirm buttons */}
      {isAwaitingConfirm ? (
        <div className="flex flex-wrap gap-x-2 gap-y-2">
          <button
            className="rounded-md bg-blue-500 px-3 py-2 text-sm text-white hover:bg-blue-600 disabled:opacity-50"
            disabled={isCurrentChatRunning}
            onClick={() => submitConfirmation(true)}
          >
            {confirmLabel}
          </button>
          <button
            className="rounded-md border px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 disabled:opacity-50"
            disabled={isCurrentChatRunning}
            onClick={() => submitConfirmation(false)}
          >
            {cancelLabel}
          </button>
        </div>
      ) : isAwaitingChoose ? (
        /* Choose options */
        <div className="flex flex-col gap-y-2">
          {chooseOptions.map((option) => (
            <button
              key={option.value}
              className="rounded-md border px-3 py-2 text-left text-sm hover:bg-gray-50 disabled:opacity-50"
              disabled={isCurrentChatRunning}
              onClick={() => submitChoice(option)}
            >
              <div className="font-medium text-gray-800">{option.label}</div>
              {option.description && (
                <div className="mt-1 text-xs text-gray-500">
                  {option.description}
                </div>
              )}
            </button>
          ))}
        </div>
      ) : (
        /* Input row with @-mention autocomplete */
        <div
          className="relative flex items-center gap-x-2"
          data-agent-input-row
        >
          <div className="relative w-full">
            <textarea
              ref={textareaRef}
              value={input}
              rows={1}
              className="w-full resize-none rounded-md border px-3 py-1.5 text-sm outline-none focus:ring-1 focus:ring-accent disabled:opacity-50"
              style={{ maxHeight: 120 }}
              placeholder={inputPlaceholder}
              disabled={isCurrentChatRunning || isAIConfigurationBlocked}
              onChange={(e) => {
                setInput(e.target.value);
                updateSelection();
              }}
              onClick={updateSelection}
              onKeyUp={updateSelection}
              onSelect={updateSelection}
              onKeyDown={handleTextareaKeydown}
            />

            {/* Mention popover */}
            {showMention && (
              <div
                ref={mentionListRef}
                className="absolute bottom-full left-0 z-50 mb-1 max-h-80 w-full overflow-y-auto rounded-md border bg-white shadow-lg"
              >
                {mentionOptions.map((option, index) => {
                  const meta = formatDomRefSuggestionMeta(option.suggestion);
                  return (
                    <div
                      key={option.value}
                      data-mention-option
                      className={`cursor-pointer px-3 py-2 ${
                        index === highlightIndex
                          ? "bg-accent/10"
                          : "hover:bg-gray-50"
                      }`}
                      onMouseDown={(e) => {
                        e.preventDefault();
                        selectMention(option);
                      }}
                      onMouseEnter={() => setHighlightIndex(index)}
                    >
                      <div className="flex flex-col text-sm">
                        <div className="flex items-center gap-x-2 text-gray-800">
                          <span className="font-medium">
                            [{option.suggestion.ref}]
                          </span>
                          <span className="truncate">
                            {option.suggestion.label}
                          </span>
                        </div>
                        {meta && (
                          <div className="mt-1 text-xs text-gray-500">
                            {meta}
                          </div>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {loading ? (
            <Button
              variant="destructive"
              size="sm"
              className="whitespace-nowrap"
              onClick={() => useAgentStore.getState().cancel(currentChat?.id)}
            >
              {t("agent.stop")}
            </Button>
          ) : (
            <Button
              size="sm"
              className="whitespace-nowrap"
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
