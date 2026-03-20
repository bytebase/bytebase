<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { runAgentLoop } from "../logic/agentLoop";
import { buildSystemPrompt } from "../logic/prompt";
import { createToolExecutor, getToolDefinitions } from "../logic/tools";
import type { Message } from "../logic/types";
import { useAgentStore } from "../store/agent";

const { t } = useI18n();
const router = useRouter();
const route = useRoute();
const agentStore = useAgentStore();
const input = ref("");

const isSendDisabled = computed(() => {
  return !input.value.trim() || agentStore.hasRunningThread;
});

const getCurrentPageSnapshot = () => ({
  path: route.fullPath,
  title: document.title,
});

async function send() {
  const text = input.value.trim();
  if (!text || agentStore.hasRunningThread) {
    return;
  }

  const page = getCurrentPageSnapshot();
  const thread = agentStore.ensureCurrentThread(page);
  const threadId = thread.id;

  agentStore.clearError(threadId);
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

  const controller = new AbortController();
  agentStore.abortController = controller;

  const systemPrompt = buildSystemPrompt(page);
  const allMessages: Message[] = [
    { role: "system", content: systemPrompt },
    ...agentStore.getMessages(threadId),
  ];

  const tools = getToolDefinitions();
  const executor = createToolExecutor(router);

  try {
    await runAgentLoop(
      allMessages,
      tools,
      executor,
      {
        onToolCall: (toolCall) => {
          const threadMessages = agentStore.getMessages(threadId);
          const lastMessage = threadMessages.at(-1);
          if (
            lastMessage?.role === "assistant" &&
            lastMessage.toolCalls &&
            !threadMessages.some(
              (message) =>
                message.role === "tool" &&
                lastMessage.toolCalls?.some((existingToolCall) => {
                  return existingToolCall.id === message.toolCallId;
                })
            )
          ) {
            agentStore.appendToolCall(threadId, lastMessage.id, toolCall);
            return;
          }

          agentStore.addMessage({
            threadId,
            role: "assistant",
            toolCalls: [toolCall],
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
    agentStore.finishRun(threadId);
  } catch (err) {
    const error = err instanceof Error ? err : new Error(String(err));
    const isAbort =
      error.name === "AbortError" || error.message.includes("[canceled]");
    if (isAbort) {
      agentStore.addMessage({
        threadId,
        role: "assistant",
        content: `_${t("agent.interrupted")}_`,
        metadata: {
          route: page.path,
        },
      });
      agentStore.finishRun(threadId);
    } else {
      agentStore.addMessage({
        threadId,
        role: "assistant",
        content: `Error: ${error.message}`,
        metadata: {
          route: page.path,
          error: error.message,
        },
      });
      agentStore.finishRun(threadId, {
        status: "error",
        lastError: error.message,
      });
    }
  } finally {
    agentStore.abortController = null;
  }
}
</script>

<template>
  <div class="border-t p-3">
    <div class="flex items-end gap-x-2">
      <textarea
        v-model="input"
        rows="1"
        :placeholder="$t('agent.input-placeholder')"
        class="flex-1 resize-none rounded-md border px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:bg-gray-50"
        :disabled="agentStore.hasRunningThread && !agentStore.loading"
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
        {{ $t("agent.send") }}
      </button>
    </div>
  </div>
</template>
