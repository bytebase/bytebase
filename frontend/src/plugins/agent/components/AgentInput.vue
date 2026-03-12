<script setup lang="ts">
import { ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { runAgentLoop } from "../logic/agentLoop";
import { buildSystemPrompt } from "../logic/prompt";
import { createToolExecutor, getToolDefinitions } from "../logic/tools";
import type { Message } from "../logic/types";
import { useAgentStore } from "../store/agent";

const router = useRouter();
const route = useRoute();
const agentStore = useAgentStore();
const input = ref("");

async function send() {
  const text = input.value.trim();
  if (!text || agentStore.loading) return;
  input.value = "";

  agentStore.addMessage({ role: "user", content: text });
  agentStore.loading = true;

  const controller = new AbortController();
  agentStore.abortController = controller;

  const systemPrompt = buildSystemPrompt({
    path: route.fullPath,
    title: document.title,
  });

  const allMessages: Message[] = [
    { role: "system", content: systemPrompt },
    ...agentStore.messages,
  ];

  const tools = getToolDefinitions();
  const executor = createToolExecutor(router);

  try {
    await runAgentLoop(
      allMessages,
      tools,
      executor,
      {
        onToolCall: (tc) => {
          // The agent loop calls onToolCall for each tool call in an assistant
          // turn. All tool calls from the same turn arrive before any
          // onToolResult. We batch them into one assistant message.
          const lastMsg = agentStore.messages[agentStore.messages.length - 1];
          if (
            lastMsg?.role === "assistant" &&
            lastMsg.toolCalls &&
            !agentStore.messages.some(
              (m) =>
                m.role === "tool" &&
                lastMsg.toolCalls!.some((t) => t.id === m.toolCallId)
            )
          ) {
            // Same turn — append to existing assistant message
            lastMsg.toolCalls.push(tc);
          } else {
            agentStore.addMessage({
              role: "assistant",
              toolCalls: [tc],
            });
          }
        },
        onToolResult: (toolCallId, result) => {
          agentStore.addMessage({
            role: "tool",
            toolCallId,
            content: result,
          });
        },
        onText: (text) => {
          agentStore.addMessage({
            role: "assistant",
            content: text,
          });
        },
      },
      controller.signal
    );
  } catch (err) {
    if ((err as Error).name !== "AbortError") {
      agentStore.addMessage({
        role: "assistant",
        content: `Error: ${(err as Error).message}`,
      });
    }
  } finally {
    agentStore.loading = false;
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
        class="flex-1 resize-none rounded-md border px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-blue-500"
        :disabled="agentStore.loading"
        @keydown.enter.exact.prevent="send"
      />
      <button
        class="rounded-md bg-blue-500 px-3 py-2 text-sm text-white hover:bg-blue-600 disabled:opacity-50"
        :disabled="!input.trim() || agentStore.loading"
        @click="send"
      >
        {{ $t("agent.send") }}
      </button>
    </div>
  </div>
</template>
