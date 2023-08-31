<template>
  <div v-for="engine in rule.engineList" :key="engine" class="relative w-4">
    <EngineIcon :engine="engineFromJSON(engine)" />
  </div>
</template>

<script lang="ts" setup>
import { RuleTemplate } from "@/types";
import { engineFromJSON } from "@/types/proto/v1/common";

defineProps<{
  rule: RuleTemplate;
}>();
</script>
