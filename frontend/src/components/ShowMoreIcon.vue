<template>
  <NTooltip trigger="hover">
    <template #trigger>
      <heroicons-outline:question-mark-circle class="w-5 h-5" v-bind="$attrs" />
    </template>
    {{ content }}
  </NTooltip>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";

withDefaults(
  defineProps<{
    content: string;
  }>(),
  {
    content: "",
  }
);
</script>
