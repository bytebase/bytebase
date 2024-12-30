<template>
  <NButton
    quaternary
    size="tiny"
    style="--n-padding: 4px"
    :disabled="disabled"
    v-bind="$attrs"
  >
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
