<template>
  <div
    class="border rounded shadow py-1 px-1 bg-gray-100 border-gray-400"
    :class="[
      message.status === 'DONE'
        ? 'w-[60%]'
        : message.status === 'FAILED'
        ? 'max-w-[60%]'
        : 'w-auto',
    ]"
  >
    <CodeView v-if="message.status === 'DONE'" :message="message" />
    <template v-else-if="message.status === 'LOADING'">
      <BBSpin class="mx-1" />
    </template>
    <template v-else>
      <div class="text-warning flex items-center gap-x-1">
        <heroicons-outline:exclaimation-triangle
          class="inline-block w-4 h-4 shrink-0"
        />
        <span class="text-sm">
          {{ message.error }}
        </span>
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import type { Message } from "../../types";
import CodeView from "./CodeView.vue";

defineProps<{
  message: Message;
}>();
</script>
