<script setup lang="ts">
import { v4 as uuidv4 } from "uuid";
import { computed, nextTick, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import type { DomRefSuggestion } from "../dom";
import { lazyExtractDomRefSuggestions } from "../dom";
import { runAgentLoop } from "../logic/agentLoop";
import { buildSystemPrompt } from "../logic/prompt";
import { createToolExecutor, getToolDefinitions } from "../logic/tools";
import type { AgentAskUserOption, Message } from "../logic/types";
import { useAgentStore } from "../store/agent";

const { t } = useI18n();
const router = useRouter();
const route = useRoute();
const agentStore = useAgentStore();
const input = ref("");
const runTokens = new Map<string, number>();

const currentThread = computed(() => agentStore.currentThread);
const currentPendingAsk = computed(() => agentStore.currentPendingAsk);
const isInterrupted = computed(() => Boolean(currentThread.value?.interrupted));
const chooseOptions = computed(() =>
  currentPendingAsk.value?.kind === "choose"
    ? (currentPendingAsk.value.options ?? [])
    : []
);
const isAwaitingConfirm = computed(
  () => currentPendingAsk.value?.kind === "confirm"
);
const isAwaitingChoose = computed(
  () =>
    currentPendingAsk.value?.kind === "choose" && chooseOptions.value.length > 0
);
const sendLabel = computed(() =>
  currentPendingAsk.value ? t("agent.reply") : t("agent.send")
);
const inputPlaceholder = computed(() => {
  if (currentPendingAsk.value?.kind === "input") {
    return currentPendingAsk.value.prompt;
  }
  if (currentPendingAsk.value?.kind === "choose") {
    return currentPendingAsk.value.prompt;
  }
  return t("agent.input-placeholder");
});
const confirmLabel = computed(
  () => currentPendingAsk.value?.confirmLabel ?? t("agent.confirm")
);
const cancelLabel = computed(
  () => currentPendingAsk.value?.cancelLabel ?? t("agent.cancel")
);
const isCurrentThreadRunning = computed(() =>
  agentStore.isThreadRunning(currentThread.value?.id)
);
const isSendDisabled = computed(() => {
  if (
    isCurrentThreadRunning.value ||
    isAwaitingConfirm.value ||
    isAwaitingChoose.value
  ) {
    return true;
  }
  return !input.value.trim();
});

const inputRef = ref<HTMLTextAreaElement>();
const domRefSuggestions = ref<DomRefSuggestion[]>([]);
const isDomRefMenuOpen = ref(false);
const activeDomRefIndex = ref(0);
const selectionStart = ref(0);
const selectionEnd = ref(0);
let domRefRequestToken = 0;

const DOM_REF_SUGGESTION_LIMIT = 8;

const normalizeSearchText = (value?: string) =>
  value?.toLowerCase().trim() ?? "";

const matchDomRefSuggestion = (suggestion: DomRefSuggestion, query: string) => {
  if (!query) {
    return true;
  }
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
  if (start !== end) {
    return null;
  }
  const prefix = text.slice(0, start);
  const matches = prefix.match(/(^|\s)@([^\s@]*)$/);
  if (!matches) {
    return null;
  }
  return {
    query: matches[2] ?? "",
    start: start - matches[2].length - 1,
    end,
  };
};

const activeDomRefQuery = computed(() =>
  getDomRefQuery(input.value, selectionStart.value, selectionEnd.value)
);

const filteredDomRefSuggestions = computed(() => {
  const query = activeDomRefQuery.value?.query ?? "";
  return domRefSuggestions.value
    .filter((suggestion: DomRefSuggestion) =>
      matchDomRefSuggestion(suggestion, query)
    )
    .slice(0, DOM_REF_SUGGESTION_LIMIT);
});

const updateSelection = () => {
  const textarea = inputRef.value;
  if (!textarea) {
    return;
  }
  selectionStart.value = textarea.selectionStart ?? 0;
  selectionEnd.value = textarea.selectionEnd ?? selectionStart.value;
};

const closeDomRefMenu = () => {
  isDomRefMenuOpen.value = false;
  activeDomRefIndex.value = 0;
};

const openDomRefMenu = () => {
  isDomRefMenuOpen.value = true;
  activeDomRefIndex.value = 0;
};

const loadDomRefSuggestions = async () => {
  const query = activeDomRefQuery.value;
  if (!query) {
    closeDomRefMenu();
    domRefSuggestions.value = [];
    return;
  }

  const requestToken = ++domRefRequestToken;
  const suggestions = await lazyExtractDomRefSuggestions();
  if (requestToken !== domRefRequestToken || !activeDomRefQuery.value) {
    return;
  }

  domRefSuggestions.value = suggestions;
  if (filteredDomRefSuggestions.value.length > 0) {
    openDomRefMenu();
    return;
  }
  closeDomRefMenu();
};

const selectDomRefSuggestion = async (suggestion: DomRefSuggestion) => {
  const query = activeDomRefQuery.value;
  const textarea = inputRef.value;
  if (!query || !textarea) {
    return;
  }

  const before = input.value.slice(0, query.start);
  const after = input.value.slice(query.end);
  const token = `[${suggestion.ref}]`;
  const suffix = after.startsWith(" ") ? "" : " ";
  const nextValue = `${before}${token}${suffix}${after}`;
  const caret = before.length + token.length + suffix.length;

  input.value = nextValue;
  closeDomRefMenu();
  await nextTick();
  textarea.focus();
  textarea.setSelectionRange(caret, caret);
  updateSelection();
};

const moveDomRefSelection = (offset: number) => {
  const total = filteredDomRefSuggestions.value.length;
  if (total === 0) {
    return;
  }
  activeDomRefIndex.value = (activeDomRefIndex.value + offset + total) % total;
};

const handleTextareaKeydown = async (event: KeyboardEvent) => {
  if (isDomRefMenuOpen.value) {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      moveDomRefSelection(1);
      return;
    }
    if (event.key === "ArrowUp") {
      event.preventDefault();
      moveDomRefSelection(-1);
      return;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      closeDomRefMenu();
      return;
    }
    if (event.key === "Enter") {
      event.preventDefault();
      const suggestion =
        filteredDomRefSuggestions.value[activeDomRefIndex.value];
      if (suggestion) {
        await selectDomRefSuggestion(suggestion);
      } else {
        closeDomRefMenu();
      }
      return;
    }
  }

  if (event.key === "Enter" && !event.shiftKey) {
    event.preventDefault();
    await send();
  }
};

const formatDomRefSuggestionMeta = (suggestion: DomRefSuggestion) => {
  return [suggestion.tag.toLowerCase(), suggestion.role]
    .filter((value): value is string => Boolean(value))
    .join(" · ");
};

const getCurrentPageSnapshot = () => ({
  path: route.fullPath,
  title: document.title,
});

const buildConversation = (
  threadId: string,
  systemPrompt: string
): Message[] => {
  return [
    { role: "system", content: systemPrompt },
    ...agentStore.getMessages(threadId),
  ];
};

const handleOutcome = (
  threadId: string,
  page: { path: string; title: string },
  errorPrefix: string,
  outcome: Awaited<ReturnType<typeof runAgentLoop>>
) => {
  switch (outcome.kind) {
    case "completed":
      agentStore.finishRun(threadId);
      return;
    case "awaiting_user":
      agentStore.awaitUser(threadId, outcome.ask);
      return;
    case "aborted":
      agentStore.interruptRun(threadId, getCurrentPageSnapshot());
      return;
    case "error":
      agentStore.addMessage({
        threadId,
        role: "assistant",
        content: `${errorPrefix}${outcome.error.message}`,
        metadata: {
          route: page.path,
          error: outcome.error.message,
        },
      });
      agentStore.finishRun(threadId, {
        status: "error",
        lastError: outcome.error.message,
      });
      return;
  }
};

async function runThread(
  threadId: string,
  page: { path: string; title: string },
  runId: string
) {
  const controller = new AbortController();
  const runToken = (runTokens.get(threadId) ?? 0) + 1;
  runTokens.set(threadId, runToken);
  agentStore.setAbortController(threadId, controller);

  const systemPrompt = buildSystemPrompt(page);
  const tools = getToolDefinitions();
  const executor = createToolExecutor(router, {
    threadId,
    onNavigate: () => {
      agentStore.updateThreadPage(threadId, getCurrentPageSnapshot());
    },
  });

  try {
    const outcome = await runAgentLoop(
      buildConversation(threadId, systemPrompt),
      tools,
      executor,
      {
        onAssistantMessage: (message) => {
          if (!message.content && !message.toolCalls?.length) {
            return;
          }
          agentStore.addMessage({
            threadId,
            role: "assistant",
            content: message.content,
            toolCalls: message.toolCalls,
            metadata: {
              route: page.path,
              runId,
            },
          });
        },
        onToolResult: (toolCallId, result) => {
          agentStore.addMessage({
            threadId,
            role: "tool",
            toolCallId,
            content: result,
            metadata: {
              route: page.path,
              runId,
            },
          });
        },
        onText: (text) => {
          if (!text) {
            return;
          }
          agentStore.addMessage({
            threadId,
            role: "assistant",
            content: text,
            metadata: {
              route: page.path,
              runId,
            },
          });
        },
      },
      controller.signal
    );

    agentStore.incrementThreadTotalTokens(
      threadId,
      outcome.totalTokensUsed ?? 0
    );

    if (runToken !== runTokens.get(threadId)) {
      return;
    }

    handleOutcome(threadId, page, "Error: ", outcome);
  } finally {
    if (agentStore.getAbortController(threadId) === controller) {
      agentStore.setAbortController(threadId, null);
    }
  }
}

async function startThreadRun(
  threadId: string,
  currentPage: { path: string; title: string }
) {
  const runId = uuidv4();

  agentStore.updateThreadPage(threadId, currentPage);
  agentStore.startRun(threadId, currentPage, {
    runId,
  });
  await runThread(threadId, currentPage, runId);
}

async function send() {
  if (agentStore.isThreadRunning(currentThread.value?.id)) {
    return;
  }

  const currentPage = getCurrentPageSnapshot();
  const thread = agentStore.ensureCurrentThread(currentPage);
  const threadId = thread.id;

  agentStore.clearError(threadId);

  if (currentPendingAsk.value) {
    const answer = input.value.trim();
    if (!answer) {
      return;
    }
    input.value = "";

    if (currentPendingAsk.value.kind === "choose") {
      agentStore.answerPendingAsk(
        threadId,
        {
          kind: "choose",
          answer,
          value: answer,
        },
        {
          route: currentPage.path,
        }
      );
    } else if (currentPendingAsk.value.kind === "input") {
      agentStore.answerPendingAsk(
        threadId,
        {
          kind: "input",
          answer,
        },
        {
          route: currentPage.path,
        }
      );
    } else {
      return;
    }
    await startThreadRun(threadId, currentPage);
    return;
  }

  const text = input.value.trim();
  if (!text) {
    return;
  }

  input.value = "";
  agentStore.addMessage({
    threadId,
    role: "user",
    content: text,
    metadata: {
      route: currentPage.path,
    },
  });
  await startThreadRun(threadId, currentPage);
}

async function retryLastTurn() {
  if (
    agentStore.isThreadRunning(currentThread.value?.id) ||
    !currentThread.value?.interrupted
  ) {
    return;
  }

  const threadId = currentThread.value.id;
  const currentPage = getCurrentPageSnapshot();
  agentStore.removeMessagesByRunId(threadId, currentThread.value.runId);
  agentStore.clearError(threadId);
  await startThreadRun(threadId, currentPage);
}

function dismissInterrupted() {
  if (!currentThread.value) {
    return;
  }
  agentStore.removeMessagesByRunId(
    currentThread.value.id,
    currentThread.value.runId
  );
  agentStore.clearError(currentThread.value.id);
}

async function submitConfirmation(confirmed: boolean) {
  const pendingAsk = currentPendingAsk.value;
  if (
    !pendingAsk ||
    pendingAsk.kind !== "confirm" ||
    agentStore.isThreadRunning(currentThread.value?.id)
  ) {
    return;
  }

  const currentPage = getCurrentPageSnapshot();
  const thread = agentStore.ensureCurrentThread(currentPage);
  const threadId = thread.id;
  const answer = confirmed ? confirmLabel.value : cancelLabel.value;

  agentStore.clearError(threadId);
  agentStore.answerPendingAsk(
    threadId,
    {
      kind: "confirm",
      answer,
      confirmed,
    },
    {
      route: currentPage.path,
    }
  );
  await startThreadRun(threadId, currentPage);
}

async function submitChoice(option: AgentAskUserOption) {
  const pendingAsk = currentPendingAsk.value;
  if (
    !pendingAsk ||
    pendingAsk.kind !== "choose" ||
    agentStore.isThreadRunning(currentThread.value?.id)
  ) {
    return;
  }

  const currentPage = getCurrentPageSnapshot();
  const thread = agentStore.ensureCurrentThread(currentPage);
  const threadId = thread.id;

  agentStore.clearError(threadId);
  agentStore.answerPendingAsk(
    threadId,
    {
      kind: "choose",
      answer: option.label,
      value: option.value,
    },
    {
      route: currentPage.path,
    }
  );
  await startThreadRun(threadId, currentPage);
}

watch(
  () => [
    agentStore.currentThreadId,
    currentPendingAsk.value?.toolCallId,
    currentPendingAsk.value?.kind,
    currentPendingAsk.value?.defaultValue,
  ],
  () => {
    if (
      currentPendingAsk.value?.kind === "input" ||
      currentPendingAsk.value?.kind === "choose"
    ) {
      input.value = currentPendingAsk.value.defaultValue ?? "";
      return;
    }
    input.value = "";
  },
  { immediate: true }
);
watch(
  () => [input.value, selectionStart.value, selectionEnd.value],
  () => {
    void loadDomRefSuggestions();
  }
);

watch(
  () => isCurrentThreadRunning.value,
  (running) => {
    if (running) {
      closeDomRefMenu();
    }
  }
);
</script>

<template>
  <div class="border-t p-3">
    <div
      v-if="currentPendingAsk"
      class="mb-3 rounded-md bg-amber-50 px-3 py-2 text-xs text-amber-700"
    >
      <div class="font-medium">{{ currentPendingAsk.prompt }}</div>
      <div class="mt-1">
        {{
          currentPendingAsk.kind === "confirm"
            ? $t("agent.pending-confirm-hint")
            : currentPendingAsk.kind === "choose"
              ? $t("agent.pending-choose-hint")
              : $t("agent.pending-input-hint")
        }}
      </div>
    </div>

    <div
      v-if="isInterrupted"
      class="mb-3 rounded-md bg-red-50 px-3 py-2 text-xs text-red-700"
    >
      <div class="font-medium">{{ $t("agent.interrupted") }}</div>
      <div class="mt-1">{{ $t("agent.interrupted-retry-hint") }}</div>
      <div class="mt-2 flex flex-wrap gap-x-2 gap-y-2">
        <button
          class="rounded-md bg-red-500 px-3 py-2 text-sm text-white hover:bg-red-600 disabled:opacity-50"
          :disabled="isCurrentThreadRunning"
          @click="retryLastTurn"
        >
          {{ $t("agent.retry-last-turn") }}
        </button>
        <button
          class="rounded-md border px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 disabled:opacity-50"
          :disabled="isCurrentThreadRunning"
          @click="dismissInterrupted"
        >
          {{ $t("common.dismiss") }}
        </button>
      </div>
    </div>

    <div v-if="isAwaitingConfirm" class="flex flex-wrap gap-x-2 gap-y-2">
      <button
        class="rounded-md bg-blue-500 px-3 py-2 text-sm text-white hover:bg-blue-600 disabled:opacity-50"
        :disabled="isCurrentThreadRunning"
        @click="submitConfirmation(true)"
      >
        {{ confirmLabel }}
      </button>
      <button
        class="rounded-md border px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 disabled:opacity-50"
        :disabled="isCurrentThreadRunning"
        @click="submitConfirmation(false)"
      >
        {{ cancelLabel }}
      </button>
    </div>

    <div v-else-if="isAwaitingChoose" class="flex flex-col gap-y-2">
      <button
        v-for="option in chooseOptions"
        :key="option.value"
        class="rounded-md border px-3 py-2 text-left text-sm hover:bg-gray-50 disabled:opacity-50"
        :disabled="isCurrentThreadRunning"
        @click="submitChoice(option)"
      >
        <div class="font-medium text-gray-800">{{ option.label }}</div>
        <div v-if="option.description" class="mt-1 text-xs text-gray-500">
          {{ option.description }}
        </div>
      </button>
    </div>

    <div v-else class="flex items-end gap-x-2">
      <div class="relative min-w-0 flex-1">
        <textarea
          ref="inputRef"
          v-model="input"
          rows="1"
          :placeholder="inputPlaceholder"
          class="block w-full resize-none rounded-md border px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:bg-gray-50"
          :disabled="isCurrentThreadRunning"
          @click="updateSelection"
          @blur="closeDomRefMenu"
          @input="updateSelection"
          @keyup="updateSelection"
          @select="updateSelection"
          @keydown.exact="handleTextareaKeydown"
        />
        <div
          v-if="isDomRefMenuOpen"
          data-testid="dom-ref-autocomplete"
          class="absolute bottom-full left-0 z-10 mb-2 w-full overflow-hidden rounded-md border bg-white shadow-lg"
        >
          <button
            v-for="(suggestion, index) in filteredDomRefSuggestions"
            :key="suggestion.ref"
            type="button"
            data-testid="dom-ref-autocomplete-item"
            class="flex w-full flex-col px-3 py-2 text-left text-sm hover:bg-gray-50"
            :class="index === activeDomRefIndex ? 'bg-blue-50' : 'bg-white'"
            @mousedown.prevent="selectDomRefSuggestion(suggestion)"
          >
            <div class="flex items-center gap-x-2 text-gray-800">
              <span class="font-medium">[{{ suggestion.ref }}]</span>
              <span class="truncate">{{ suggestion.label }}</span>
            </div>
            <div
              v-if="formatDomRefSuggestionMeta(suggestion)"
              class="mt-1 text-xs text-gray-500"
            >
              {{ formatDomRefSuggestionMeta(suggestion) }}
            </div>
          </button>
        </div>
      </div>
      <button
        v-if="agentStore.loading"
        class="rounded-md bg-red-500 px-3 py-2 text-sm text-white hover:bg-red-600"
        @click="agentStore.cancel(currentThread?.id)"
      >
        {{ $t("agent.stop") }}
      </button>
      <button
        v-else
        class="rounded-md bg-blue-500 px-3 py-2 text-sm text-white hover:bg-blue-600 disabled:opacity-50"
        :disabled="isSendDisabled"
        @click="send"
      >
        {{ sendLabel }}
      </button>
    </div>
  </div>
</template>
