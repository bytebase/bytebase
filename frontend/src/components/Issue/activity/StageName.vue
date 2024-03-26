<template>
  <span class="font-medium text-main">
    {{ stage.name }}
  </span>
</template>

<script lang="ts" setup>
import type { Issue, Stage } from "@/types";

defineProps<{
  issue: Issue;
  stage: Stage;
}>();
</script>
