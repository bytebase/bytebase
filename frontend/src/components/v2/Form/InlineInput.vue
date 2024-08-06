<template>
  <NInput
    v-if="!disabled"
    :value="value"
    :style="style"
    v-bind="$attrs"
    @focus="focused = true"
    @blur="focused = false"
    @update:value="$emit('update:value', $event)"
  />
  <NPerformantEllipsis
    v-else
    style="padding: 0 var(--n-padding-right) 0 var(--n-padding-left)"
    v-bind="$attrs"
  >
    {{ value }}
  </NPerformantEllipsis>
</template>

<script setup lang="ts">
import { NInput, NPerformantEllipsis } from "naive-ui";
import type { CSSProperties } from "vue";
import { computed, ref } from "vue";

defineOptions({
  inheritAttrs: false,
});

defineProps<{
  value: string | undefined | null;
  disabled?: boolean;
}>();
defineEmits<{
  (event: "update:value", value: string): void;
}>();

const focused = ref(false);

const style = computed(() => {
  const style: CSSProperties = {
    cursor: "default",
    "--n-color": "transparent",
    "--n-color-disabled": "transparent",
  };
  const border = focused.value
    ? "1px solid rgb(var(--color-control-border))"
    : "none";
  style["--n-border"] = border;
  style["--n-border-disabled"] = border;

  return style;
});
</script>
