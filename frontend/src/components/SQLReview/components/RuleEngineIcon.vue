<template>
  <EngineIcon :engine="engineFromJSON(engine)" />
</template>

<script lang="ts" setup>
import { SchemaRuleEngineType } from "@/types";
import { engineFromJSON } from "@/types/proto/v1/common";

defineProps<{
  engine: SchemaRuleEngineType;
}>();
</script>
