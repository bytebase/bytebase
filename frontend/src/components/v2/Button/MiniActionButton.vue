<template>
  <NButton
    quaternary
    size="tiny"
    :type="type"
    style="--n-padding: 4px"
    :disabled="disabled"
    v-bind="$attrs"
    @click="$emit('click', $event)"
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
  type?:
    | "default"
    | "tertiary"
    | "primary"
    | "success"
    | "info"
    | "warning"
    | "error";
}>();

defineEmits<{
  (event: "click", e: MouseEvent): void;
}>();
</script>
