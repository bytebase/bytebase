<template>
  <NTag :size="size" round type="info">
    {{ $t("settings.members.service-account") }}
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
