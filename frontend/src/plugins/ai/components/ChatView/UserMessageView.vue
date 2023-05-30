<template>
  <div
    class="max-w-[60%] border rounded shadow py-1 px-2 bg-indigo-100 border-indigo-400"
  >
    {{ message.content }}
  </div>
</template>

<script lang="ts" setup>
import type { Message } from "../../types";

defineProps<{
  message: Message;
}>();
</script>
