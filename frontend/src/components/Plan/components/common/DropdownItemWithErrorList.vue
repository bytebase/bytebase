<template>
  <NTooltip
    :disabled="errors.length === 0"
    :keep-alive-on-hover="false"
    style="width: max-content"
  >
    <template #trigger>
      <slot />
    </template>
    <template #default>
      <ErrorList :errors="errors" />
    </template>
  </NTooltip>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import ErrorList from "./ErrorList.vue";

defineProps<{
  errors: string[];
}>();
</script>
