<template>
  <NSpin
    v-bind="$attrs"
    :size="size"
    :stroke-width="25"
    :class="size === 'small' && 'w-5 h-5'"
  />
</template>

<script lang="ts" setup>
import { NSpin } from "naive-ui";

withDefaults(
  defineProps<{
    size?: "small" | "medium" | "large";
  }>(),
  {
    size: "small",
  }
);
</script>
