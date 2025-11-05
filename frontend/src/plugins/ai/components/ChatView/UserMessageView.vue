<template>
  <div
    class="user-message max-w-[60%] min-w-36 border rounded-sm shadow-sm py-1 px-2 bg-indigo-100 border-indigo-400"
  >
    <Markdown
      :content="message.content"
      :code-block-props="{
        width: 0.6,
      }"
    />
  </div>
</template>

<script lang="ts" setup>
import type { Message } from "../../types";
import Markdown from "./Markdown";

defineProps<{
  message: Message;
}>();
</script>
