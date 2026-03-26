<script setup lang="ts">
import { type InputInst, NButton, NInput } from "naive-ui";
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

const currentChat = computed(() => agentStore.currentChat);
const currentPendingAsk = computed(() => agentStore.currentPendingAsk);
const isInterrupted = computed(() => Boolean(currentChat.value?.interrupted));
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
const isCurrentChatRunning = computed(() =>
  agentStore.isChatRunning(currentChat.value?.id)
);
const isSendDisabled = computed(() => {
  if (
    isCurrentChatRunning.value ||
    isAwaitingConfirm.value ||
    isAwaitingChoose.value
  ) {
    return true;
  }
  return !input.value.trim();
});

const inputInstRef = ref<InputInst | null>(null);
const getTextareaEl = () => inputInstRef.value?.textareaElRef ?? null;
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
  return domRefSuggestions.value.filter((suggestion: DomRefSuggestion) =>
    matchDomRefSuggestion(suggestion, query)
  );
});
const domRefSuggestionPageStart = computed(
  () =>
    Math.floor(activeDomRefIndex.value / DOM_REF_SUGGESTION_LIMIT) *
    DOM_REF_SUGGESTION_LIMIT
);
const visibleDomRefSuggestions = computed(() =>
  filteredDomRefSuggestions.value.slice(
    domRefSuggestionPageStart.value,
    domRefSuggestionPageStart.value + DOM_REF_SUGGESTION_LIMIT
  )
);
const activeVisibleDomRefIndex = computed(
  () => activeDomRefIndex.value - domRefSuggestionPageStart.value
);

const updateSelection = () => {
  const textarea = getTextareaEl();
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
  const textarea = getTextareaEl();
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

  if (
    event.key === "Enter" &&
    !event.shiftKey &&
    !event.altKey &&
    !event.ctrlKey &&
    !event.metaKey
  ) {
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

const buildChatHistory = (chatId: string, systemPrompt: string): Message[] => {
  return [
    { role: "system", content: systemPrompt },
    ...agentStore.getMessages(chatId),
  ];
};

const handleOutcome = (
  chatId: string,
  page: { path: string; title: string },
  errorPrefix: string,
  outcome: Awaited<ReturnType<typeof runAgentLoop>>
) => {
  switch (outcome.kind) {
    case "completed":
      agentStore.finishChatRun(chatId);
      return;
    case "awaiting_user":
      agentStore.awaitUser(chatId, outcome.ask);
      return;
    case "aborted":
      agentStore.interruptChatRun(chatId, getCurrentPageSnapshot());
      return;
    case "error":
      agentStore.addMessage({
        chatId,
        role: "assistant",
        content: `${errorPrefix}${outcome.error.message}`,
        metadata: {
          route: page.path,
          error: outcome.error.message,
        },
      });
      agentStore.finishChatRun(chatId, {
        status: "error",
        lastError: outcome.error.message,
      });
      return;
  }
};

async function runChat(
  chatId: string,
  page: { path: string; title: string },
  runId: string
) {
  const controller = new AbortController();
  const runToken = (runTokens.get(chatId) ?? 0) + 1;
  runTokens.set(chatId, runToken);
  agentStore.setAbortController(chatId, controller);

  const systemPrompt = buildSystemPrompt(page);
  const tools = getToolDefinitions();
  const executor = createToolExecutor(router, {
    chatId,
    onNavigate: () => {
      agentStore.updateChatPage(chatId, getCurrentPageSnapshot());
    },
  });

  try {
    const outcome = await runAgentLoop(
      buildChatHistory(chatId, systemPrompt),
      tools,
      executor,
      {
        onAssistantMessage: (message) => {
          if (!message.content && !message.toolCalls?.length) {
            return;
          }
          agentStore.addMessage({
            chatId,
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
            chatId,
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
            chatId,
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

    agentStore.incrementChatTotalTokens(chatId, outcome.totalTokensUsed ?? 0);

    if (runToken !== runTokens.get(chatId)) {
      return;
    }

    handleOutcome(chatId, page, "Error: ", outcome);
  } finally {
    if (agentStore.getAbortController(chatId) === controller) {
      agentStore.setAbortController(chatId, null);
    }
  }
}

async function startChatRun(
  chatId: string,
  currentPage: { path: string; title: string }
) {
  const runId = uuidv4();

  agentStore.updateChatPage(chatId, currentPage);
  agentStore.startChatRun(chatId, currentPage, {
    runId,
  });
  await runChat(chatId, currentPage, runId);
}

async function send() {
  if (agentStore.isChatRunning(currentChat.value?.id)) {
    return;
  }

  const currentPage = getCurrentPageSnapshot();
  const thread = agentStore.ensureCurrentChat(currentPage);
  const chatId = thread.id;

  agentStore.clearError(chatId);

  if (currentPendingAsk.value) {
    const answer = input.value.trim();
    if (!answer) {
      return;
    }
    input.value = "";

    if (currentPendingAsk.value.kind === "choose") {
      agentStore.answerPendingAsk(
        chatId,
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
        chatId,
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
    await startChatRun(chatId, currentPage);
    return;
  }

  const text = input.value.trim();
  if (!text) {
    return;
  }

  input.value = "";
  agentStore.addMessage({
    chatId,
    role: "user",
    content: text,
    metadata: {
      route: currentPage.path,
    },
  });
  await startChatRun(chatId, currentPage);
}

async function retryLastTurn() {
  if (
    agentStore.isChatRunning(currentChat.value?.id) ||
    !currentChat.value?.interrupted
  ) {
    return;
  }

  const chatId = currentChat.value.id;
  const currentPage = getCurrentPageSnapshot();
  agentStore.removeMessagesByRunId(chatId, currentChat.value.runId);
  agentStore.clearError(chatId);
  await startChatRun(chatId, currentPage);
}

function dismissInterrupted() {
  if (!currentChat.value) {
    return;
  }
  agentStore.removeMessagesByRunId(
    currentChat.value.id,
    currentChat.value.runId
  );
  agentStore.clearError(currentChat.value.id);
}

async function submitConfirmation(confirmed: boolean) {
  const pendingAsk = currentPendingAsk.value;
  if (
    !pendingAsk ||
    pendingAsk.kind !== "confirm" ||
    agentStore.isChatRunning(currentChat.value?.id)
  ) {
    return;
  }

  const currentPage = getCurrentPageSnapshot();
  const thread = agentStore.ensureCurrentChat(currentPage);
  const chatId = thread.id;
  const answer = confirmed ? confirmLabel.value : cancelLabel.value;

  agentStore.clearError(chatId);
  agentStore.answerPendingAsk(
    chatId,
    {
      kind: "confirm",
      answer,
      confirmed,
    },
    {
      route: currentPage.path,
    }
  );
  await startChatRun(chatId, currentPage);
}

async function submitChoice(option: AgentAskUserOption) {
  const pendingAsk = currentPendingAsk.value;
  if (
    !pendingAsk ||
    pendingAsk.kind !== "choose" ||
    agentStore.isChatRunning(currentChat.value?.id)
  ) {
    return;
  }

  const currentPage = getCurrentPageSnapshot();
  const thread = agentStore.ensureCurrentChat(currentPage);
  const chatId = thread.id;

  agentStore.clearError(chatId);
  agentStore.answerPendingAsk(
    chatId,
    {
      kind: "choose",
      answer: option.label,
      value: option.value,
    },
    {
      route: currentPage.path,
    }
  );
  await startChatRun(chatId, currentPage);
}

watch(
  () => [
    agentStore.currentChatId,
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
  () => filteredDomRefSuggestions.value.length,
  (length) => {
    if (length === 0) {
      activeDomRefIndex.value = 0;
      return;
    }
    if (activeDomRefIndex.value >= length) {
      activeDomRefIndex.value = length - 1;
    }
  }
);

watch(
  () => isCurrentChatRunning.value,
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
          :disabled="isCurrentChatRunning"
          @click="retryLastTurn"
        >
          {{ $t("agent.retry-last-chat-turn") }}
        </button>
        <button
          class="rounded-md border px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 disabled:opacity-50"
          :disabled="isCurrentChatRunning"
          @click="dismissInterrupted"
        >
          {{ $t("common.dismiss") }}
        </button>
      </div>
    </div>

    <div v-if="isAwaitingConfirm" class="flex flex-wrap gap-x-2 gap-y-2">
      <button
        class="rounded-md bg-blue-500 px-3 py-2 text-sm text-white hover:bg-blue-600 disabled:opacity-50"
        :disabled="isCurrentChatRunning"
        @click="submitConfirmation(true)"
      >
        {{ confirmLabel }}
      </button>
      <button
        class="rounded-md border px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 disabled:opacity-50"
        :disabled="isCurrentChatRunning"
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
        :disabled="isCurrentChatRunning"
        @click="submitChoice(option)"
      >
        <div class="font-medium text-gray-800">{{ option.label }}</div>
        <div v-if="option.description" class="mt-1 text-xs text-gray-500">
          {{ option.description }}
        </div>
      </button>
    </div>

    <div v-else class="relative" data-agent-input-row>
      <NInput
        ref="inputInstRef"
        v-model:value="input"
        type="textarea"
        size="small"
        class="w-full"
        :autosize="{
          minRows: 1,
          maxRows: 6,
        }"
        :resizable="false"
        :placeholder="inputPlaceholder"
        :disabled="isCurrentChatRunning"
        :input-props="{
          onSelect: updateSelection,
        }"
        @blur="closeDomRefMenu"
        @click="updateSelection"
        @input="updateSelection"
        @keyup="updateSelection"
        @keydown="handleTextareaKeydown"
      >
        <template #suffix>
          <NButton
            v-if="agentStore.loading"
            type="error"
            size="small"
            class="whitespace-nowrap"
            @click="agentStore.cancel(currentChat?.id)"
          >
            {{ $t("agent.stop") }}
          </NButton>
          <NButton
            v-else
            type="primary"
            size="small"
            class="whitespace-nowrap"
            :disabled="isSendDisabled"
            @click="send"
          >
            {{ sendLabel }}
          </NButton>
        </template>
      </NInput>
      <div
        v-if="isDomRefMenuOpen"
        data-testid="dom-ref-autocomplete"
        class="absolute bottom-full left-0 z-10 mb-2 w-full overflow-hidden rounded-md border bg-white shadow-lg"
      >
        <button
          v-for="(suggestion, index) in visibleDomRefSuggestions"
          :key="suggestion.ref"
          type="button"
          data-testid="dom-ref-autocomplete-item"
          class="flex w-full flex-col px-3 py-2 text-left text-sm hover:bg-gray-50"
          :class="index === activeVisibleDomRefIndex ? 'bg-blue-50' : 'bg-white'"
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
  </div>
</template>
