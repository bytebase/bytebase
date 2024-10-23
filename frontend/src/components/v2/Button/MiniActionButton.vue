<template>
  <NButton quaternary size="tiny" style="--n-padding: 4px" :disabled="disabled">
    <template #icon>
      <slot />
    </template>
    <template #default>
      <slot name="text" />
    </template>
  </NButton>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";

defineProps<{
  disabled?: boolean;
}>();
</script>
