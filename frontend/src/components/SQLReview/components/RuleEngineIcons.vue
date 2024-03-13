<template>
  <div v-for="engine in rule.engineList" :key="engine" class="relative w-4">
    <RuleEngineIcon :engine="engine" />
  </div>
</template>

<script lang="ts" setup>
import { RuleTemplate } from "@/types";

defineProps<{
  rule: RuleTemplate;
}>();
</script>
