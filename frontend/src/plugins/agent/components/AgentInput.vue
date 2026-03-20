<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
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

const currentPendingAsk = computed(() => agentStore.currentPendingAsk);
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
const isSendDisabled = computed(() => {
  if (
    agentStore.hasRunningThread ||
    isAwaitingConfirm.value ||
    isAwaitingChoose.value
  ) {
    return true;
  }
  return !input.value.trim();
});

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
      agentStore.addMessage({
        threadId,
        role: "assistant",
        content: `_${t("agent.interrupted")}_`,
        metadata: {
          route: page.path,
        },
      });
      agentStore.finishRun(threadId);
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
  page: { path: string; title: string }
) {
  const controller = new AbortController();
  agentStore.abortController = controller;

  const systemPrompt = buildSystemPrompt(page);
  const tools = getToolDefinitions();
  const executor = createToolExecutor(router);

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
            },
          });
        },
      },
      controller.signal
    );

    handleOutcome(threadId, page, "Error: ", outcome);
  } finally {
    agentStore.abortController = null;
  }
}

async function send() {
  if (agentStore.hasRunningThread) {
    return;
  }

  const page = getCurrentPageSnapshot();
  const thread = agentStore.ensureCurrentThread(page);
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
          route: page.path,
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
          route: page.path,
        }
      );
    } else {
      return;
    }
    agentStore.startRun(threadId, page);
    await runThread(threadId, page);
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
      route: page.path,
    },
  });
  agentStore.startRun(threadId, page);
  await runThread(threadId, page);
}

async function submitConfirmation(confirmed: boolean) {
  const pendingAsk = currentPendingAsk.value;
  if (
    !pendingAsk ||
    pendingAsk.kind !== "confirm" ||
    agentStore.hasRunningThread
  ) {
    return;
  }

  const page = getCurrentPageSnapshot();
  const thread = agentStore.ensureCurrentThread(page);
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
      route: page.path,
    }
  );
  agentStore.startRun(threadId, page);
  await runThread(threadId, page);
}

async function submitChoice(option: AgentAskUserOption) {
  const pendingAsk = currentPendingAsk.value;
  if (
    !pendingAsk ||
    pendingAsk.kind !== "choose" ||
    agentStore.hasRunningThread
  ) {
    return;
  }

  const page = getCurrentPageSnapshot();
  const thread = agentStore.ensureCurrentThread(page);
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
      route: page.path,
    }
  );
  agentStore.startRun(threadId, page);
  await runThread(threadId, page);
}

watch(
  () => currentPendingAsk.value?.toolCallId,
  () => {
    if (currentPendingAsk.value?.kind === "input") {
      input.value = currentPendingAsk.value.defaultValue ?? "";
      return;
    }
    if (currentPendingAsk.value?.kind === "choose") {
      input.value = currentPendingAsk.value.defaultValue ?? "";
      return;
    }
    if (currentPendingAsk.value?.kind === "confirm") {
      input.value = "";
    }
  },
  { immediate: true }
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

    <div v-if="isAwaitingConfirm" class="flex flex-wrap gap-x-2 gap-y-2">
      <button
        class="rounded-md bg-blue-500 px-3 py-2 text-sm text-white hover:bg-blue-600 disabled:opacity-50"
        :disabled="agentStore.hasRunningThread"
        @click="submitConfirmation(true)"
      >
        {{ confirmLabel }}
      </button>
      <button
        class="rounded-md border px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 disabled:opacity-50"
        :disabled="agentStore.hasRunningThread"
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
        :disabled="agentStore.hasRunningThread"
        @click="submitChoice(option)"
      >
        <div class="font-medium text-gray-800">{{ option.label }}</div>
        <div v-if="option.description" class="mt-1 text-xs text-gray-500">
          {{ option.description }}
        </div>
      </button>
    </div>

    <div v-else class="flex items-end gap-x-2">
      <textarea
        v-model="input"
        rows="1"
        :placeholder="inputPlaceholder"
        class="flex-1 resize-none rounded-md border px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:bg-gray-50"
        :disabled="agentStore.hasRunningThread"
        @keydown.enter.exact.prevent="send"
      />
      <button
        v-if="agentStore.loading"
        class="rounded-md bg-red-500 px-3 py-2 text-sm text-white hover:bg-red-600"
        @click="agentStore.cancel()"
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
