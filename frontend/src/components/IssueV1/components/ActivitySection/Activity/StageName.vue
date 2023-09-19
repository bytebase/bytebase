<template>
  <span class="font-medium text-main">
    {{ stage.title }}
  </span>
</template>

<script lang="ts" setup>
import { ComposedIssue } from "@/types";
import { Stage } from "@/types/proto/v1/rollout_service";

defineProps<{
  issue: ComposedIssue;
  stage: Stage;
}>();
</script>
