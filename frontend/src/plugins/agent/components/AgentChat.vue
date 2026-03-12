<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import { useAgentStore } from "../store/agent";
import ToolCallCard from "./ToolCallCard.vue";

const agentStore = useAgentStore();
const chatContainer = ref<HTMLElement | null>(null);

const displayMessages = computed(() =>
  agentStore.messages.filter(
    (msg) => msg.role === "user" || msg.role === "assistant"
  )
);

function getToolResult(
  displayIndex: number,
  toolCallId: string
): string | undefined {
  // Map display index back to the full messages array
  const msg = displayMessages.value[displayIndex];
  if (!msg) return undefined;

  const fullIndex = agentStore.messages.indexOf(msg);
  if (fullIndex < 0) return undefined;

  // Look forward in the full messages array for the tool result
  for (let i = fullIndex + 1; i < agentStore.messages.length; i++) {
    const m = agentStore.messages[i];
    if (m.role === "tool" && m.toolCallId === toolCallId) {
      return m.content;
    }
    // Stop if we hit another assistant message (new turn)
    if (m.role === "assistant" && m.content && !m.toolCalls?.length) {
      break;
    }
  }
  return undefined;
}

watch(
  () => agentStore.messages.length,
  async () => {
    await nextTick();
    if (chatContainer.value) {
      const el = chatContainer.value;
      el.scrollTop = el.scrollHeight;
    }
  }
);
</script>

<template>
  <div ref="chatContainer" class="overflow-y-auto p-3 space-y-3">
    <template v-for="(msg, i) in displayMessages" :key="i">
      <!-- User message -->
      <div v-if="msg.role === 'user'" class="flex justify-end">
        <div class="max-w-[80%] rounded-lg px-3 py-2 bg-blue-50 text-sm">
          {{ msg.content }}
        </div>
      </div>
      <!-- Assistant message -->
      <div v-else-if="msg.role === 'assistant'" class="flex flex-col gap-y-2">
        <div
          v-if="msg.content"
          class="max-w-[80%] rounded-lg px-3 py-2 bg-gray-50 text-sm whitespace-pre-wrap"
        >
          {{ msg.content }}
        </div>
        <ToolCallCard
          v-for="tc in msg.toolCalls"
          :key="tc.id"
          :tool-call="tc"
          :result="getToolResult(i, tc.id)"
        />
      </div>
    </template>
    <!-- Loading -->
    <div
      v-if="agentStore.loading"
      class="flex items-center gap-x-2 text-sm text-gray-400"
    >
      <span class="animate-pulse">&#9679;</span> Thinking...
    </div>
  </div>
</template>
