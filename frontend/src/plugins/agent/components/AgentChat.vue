<script setup lang="ts">
import { NButton } from "naive-ui";
import remarkGfm from "remark-gfm";
import remarkParse from "remark-parse";
import { unified } from "unified";
import { computed, nextTick, ref, watch } from "vue";
import { useRouter } from "vue-router";
import AstToMarkdown from "@/plugins/ai/components/ChatView/Markdown/AstToVNode.vue";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import { hasWorkspacePermissionV2 } from "@/utils";
import type { AgentMessage } from "../logic/types";
import { useAgentStore } from "../store/agent";
import ToolCallCard from "./ToolCallCard.vue";

const markdownProcessor = unified().use(remarkParse).use(remarkGfm);

const parseMarkdown = (content: string) => {
  return markdownProcessor.parse(content ?? "");
};

const router = useRouter();
const agentStore = useAgentStore();
const chatContainer = ref<HTMLElement | null>(null);

const allowConfigure = computed(() =>
  hasWorkspacePermissionV2("bb.settings.set")
);
const showAIConfigurationRecovery = computed(
  () => agentStore.currentChatRequiresAIConfiguration
);
const displayMessages = computed(() =>
  agentStore.messages.filter(
    (message): message is AgentMessage =>
      message.role === "user" || message.role === "assistant"
  )
);

function getToolResult(
  messageId: string,
  toolCallId: string
): string | undefined {
  const fullIndex = agentStore.messages.findIndex(
    (message) => message.id === messageId
  );
  if (fullIndex < 0) {
    return undefined;
  }

  for (let index = fullIndex + 1; index < agentStore.messages.length; index++) {
    const message = agentStore.messages[index];
    if (message.role === "tool" && message.toolCallId === toolCallId) {
      return message.content;
    }
    if (
      message.role === "assistant" &&
      message.content &&
      !message.toolCalls?.length
    ) {
      break;
    }
  }
  return undefined;
}

function goConfigure() {
  agentStore.clearError(agentStore.currentChatId);
  router.push({
    name: SETTING_ROUTE_WORKSPACE_GENERAL,
    hash: "#ai-assistant",
  });
}

function dismissAIConfigurationRecovery() {
  agentStore.clearError(agentStore.currentChatId);
}
watch(
  [() => agentStore.currentChatId, () => agentStore.messages.length],
  async () => {
    await nextTick();
    if (chatContainer.value) {
      chatContainer.value.scrollTop = chatContainer.value.scrollHeight;
    }
  }
);
</script>

<template>
  <div ref="chatContainer" class="overflow-y-auto space-y-3 p-3">
    <template v-for="msg in displayMessages" :key="msg.id">
      <div v-if="msg.role === 'user'" class="flex justify-end">
        <div class="max-w-[80%] rounded-lg bg-blue-50 px-3 py-2 text-sm">
          {{ msg.content }}
        </div>
      </div>
      <div v-else class="flex flex-col gap-y-2">
        <div
          v-if="msg.content"
          class="max-w-[80%] rounded-lg bg-gray-50 px-3 py-2 text-sm markdown-content"
        >
          <AstToMarkdown :ast="parseMarkdown(msg.content)" />
        </div>
        <ToolCallCard
          v-for="toolCall in msg.toolCalls"
          :key="toolCall.id"
          :tool-call="toolCall"
          :result="getToolResult(msg.id, toolCall.id)"
        />
      </div>
    </template>
    <div
      v-if="agentStore.loading"
      class="flex items-center gap-x-2 text-sm text-gray-400"
    >
      <span class="animate-pulse">&#9679;</span> {{ $t("common.loading") }}
    </div>
    <div
      v-if="showAIConfigurationRecovery"
      class="max-w-[80%] rounded-lg border border-amber-200 bg-amber-50 px-3 py-3 text-sm text-amber-900"
    >
      <div class="font-medium">
        {{ $t("agent.ai-not-configured.title") }}
      </div>
      <div class="mt-1">
        {{ $t("agent.ai-not-configured.description") }}
      </div>
      <div class="mt-3 flex flex-wrap gap-x-2 gap-y-2">
        <NButton v-if="allowConfigure" type="primary" size="small" @click="goConfigure">
          {{ $t("agent.ai-not-configured.configure") }}
        </NButton>
        <NButton size="small" @click="dismissAIConfigurationRecovery">
          {{ $t("common.dismiss") }}
        </NButton>
      </div>
      <div v-if="!allowConfigure" class="mt-1 text-amber-800">
        {{ $t("agent.ai-not-configured.contact-admin") }}
      </div>
    </div>
    <div
      v-else-if="agentStore.error"
      class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600"
    >
      {{ agentStore.error }}
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
  @apply my-1 overflow-x-auto rounded bg-gray-100 p-2 text-xs;
}
.markdown-content :deep(code) {
  @apply rounded bg-gray-200 px-1 text-xs;
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
  @apply my-1 font-semibold;
}
.markdown-content :deep(a) {
  @apply text-blue-600 underline;
}
.markdown-content :deep(blockquote) {
  @apply my-1 border-l-2 border-gray-300 pl-2 text-gray-600;
}
.markdown-content :deep(table) {
  @apply my-1 border-collapse text-xs;
}
.markdown-content :deep(th),
.markdown-content :deep(td) {
  @apply border border-gray-300 px-2 py-1;
}
</style>
