<template>
  <NTag :size="size" v-bind="$attrs" round type="success">
    {{ $t("common.you") }}
  </NTag>
</template>

<script lang="ts" setup>
import { NTag } from "naive-ui";

withDefaults(
  defineProps<{
    size?: "tiny" | "small" | "medium" | "large";
  }>(),
  {
    size: "small",
  }
);
</script>
