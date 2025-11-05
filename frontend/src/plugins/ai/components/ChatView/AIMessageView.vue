<template>
  <div
    class="border rounded-sm shadow-sm py-1 px-1 bg-gray-50 border-gray-400"
    :class="[
      message.status === 'DONE'
        ? 'w-full min-w-36'
        : message.status === 'FAILED'
          ? 'max-w-[40%] min-w-36'
          : 'w-auto',
    ]"
  >
    <Markdown
      v-if="message.status === 'DONE'"
      :content="message.content"
      :code-block-props="{
        width: 1.0,
      }"
    />
    <div v-else-if="message.status === 'LOADING'" class="flex items-center">
      <BBSpin class="mx-1" :size="18" />
    </div>
    <template v-else>
      <div class="text-warning flex items-center gap-x-1">
        <TriangleAlertIcon class="inline-block w-4 h-4 shrink-0" />
        <span class="text-sm">
          {{ message.error }}
        </span>
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { TriangleAlertIcon } from "lucide-vue-next";
import { BBSpin } from "@/bbkit";
import type { Message } from "../../types";
import Markdown from "./Markdown";

defineProps<{
  message: Message;
}>();
</script>
