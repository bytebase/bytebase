<script setup lang="ts">
import remarkGfm from "remark-gfm";
import remarkParse from "remark-parse";
import { unified } from "unified";
import { computed, nextTick, ref, watch } from "vue";
import AstToMarkdown from "@/plugins/ai/components/ChatView/Markdown/AstToVNode.vue";
import { useAgentStore } from "../store/agent";
import ToolCallCard from "./ToolCallCard.vue";

const markdownProcessor = unified().use(remarkParse).use(remarkGfm);

const parseMarkdown = (content: string) => {
  return markdownProcessor.parse(content ?? "");
};

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
          class="max-w-[80%] rounded-lg px-3 py-2 bg-gray-50 text-sm markdown-content"
        >
          <AstToMarkdown :ast="parseMarkdown(msg.content)" />
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

<style scoped lang="postcss">
.markdown-content :deep(p) {
  @apply my-1;
}
.markdown-content :deep(p:first-child) {
  @apply mt-0;
}
.markdown-content :deep(p:last-child) {
  @apply mb-0;
}
.markdown-content :deep(pre) {
  @apply my-1 p-2 bg-gray-100 rounded text-xs overflow-x-auto;
}
.markdown-content :deep(code) {
  @apply bg-gray-200 px-1 rounded text-xs;
}
.markdown-content :deep(pre code) {
  @apply bg-transparent px-0;
}
.markdown-content :deep(ul),
.markdown-content :deep(ol) {
  @apply my-1 pl-5;
}
.markdown-content :deep(ul) {
  @apply list-disc;
}
.markdown-content :deep(ol) {
  @apply list-decimal;
}
.markdown-content :deep(li) {
  @apply my-0.5;
}
.markdown-content :deep(h1),
.markdown-content :deep(h2),
.markdown-content :deep(h3) {
  @apply font-semibold my-1;
}
.markdown-content :deep(a) {
  @apply text-blue-600 underline;
}
.markdown-content :deep(blockquote) {
  @apply border-l-2 border-gray-300 pl-2 my-1 text-gray-600;
}
.markdown-content :deep(table) {
  @apply my-1 border-collapse text-xs;
}
.markdown-content :deep(th),
.markdown-content :deep(td) {
  @apply border border-gray-300 px-2 py-1;
}
</style>
