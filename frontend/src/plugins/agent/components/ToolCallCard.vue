<script setup lang="ts">
import { ref } from "vue";
import type { ToolCall } from "../logic/types";

defineProps<{
  toolCall: ToolCall;
  result?: string;
}>();

const expanded = ref(false);

function formatJson(value: string): string {
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}
</script>

<template>
  <div class="rounded border bg-gray-50 text-xs">
    <div
      class="flex items-center gap-x-2 px-2 py-1.5 cursor-pointer"
      @click="expanded = !expanded"
    >
      <span class="font-mono text-gray-600">{{ toolCall.name }}</span>
      <span v-if="result" class="text-green-500">&#10003;</span>
      <span v-else class="animate-pulse text-gray-400">&#9679;</span>
      <span class="ml-auto text-gray-400">{{ expanded ? "\u25BE" : "\u25B8" }}</span>
    </div>
    <div v-if="expanded" class="border-t px-2 py-1.5 space-y-1">
      <div class="text-gray-500">Args:</div>
      <pre class="text-gray-700 whitespace-pre-wrap break-all">{{
        formatJson(toolCall.arguments)
      }}</pre>
      <template v-if="result">
        <div class="text-gray-500">Result:</div>
        <pre
          class="text-gray-700 whitespace-pre-wrap break-all max-h-32 overflow-y-auto"
          >{{ formatJson(result) }}</pre
        >
      </template>
    </div>
  </div>
</template>
