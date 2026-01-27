<template>
  <NTag :size="size" round type="warning">
    {{ $t("settings.members.workload-identity") }}
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
