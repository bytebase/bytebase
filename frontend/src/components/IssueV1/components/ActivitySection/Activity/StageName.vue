<template>
  <span class="font-medium text-main">
    {{ stage.title }}
  </span>
</template>

<script lang="ts" setup>
import type { ComposedIssue } from "@/types";
import type { Stage } from "@/types/proto/v1/rollout_service";

defineProps<{
  issue: ComposedIssue;
  stage: Stage;
}>();
</script>
