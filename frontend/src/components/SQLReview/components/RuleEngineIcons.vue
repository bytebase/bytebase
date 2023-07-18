<template>
  <EngineIcon
    v-for="engine in rule.engineList"
    :key="engine"
    :engine="engineFromJSON(engine)"
  />
</template>

<script lang="ts" setup>
import { RuleTemplate } from "@/types";
import { engineFromJSON } from "@/types/proto/v1/common";

defineProps<{
  rule: RuleTemplate;
}>();
</script>
