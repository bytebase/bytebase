<template>
  <RuleEngineIcon
    v-for="engine in rule.engineList"
    :key="engine"
    :engine="engine"
  />
</template>

<script lang="ts" setup>
import { RuleTemplate } from "@/types";
import RuleEngineIcon from "./RuleEngineIcon.vue";

defineProps<{
  rule: RuleTemplate;
}>();
</script>
