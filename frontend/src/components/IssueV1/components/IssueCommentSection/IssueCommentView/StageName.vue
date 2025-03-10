<template>
  <span class="font-medium text-main">
    {{ extractEnvironmentResourceName(stage.environment) }}
  </span>
</template>

<script lang="ts" setup>
import type { ComposedIssue } from "@/types";
import type { Stage } from "@/types/proto/v1/rollout_service";
import { extractEnvironmentResourceName } from "@/utils";

defineProps<{
  issue: ComposedIssue;
  stage: Stage;
}>();
</script>
