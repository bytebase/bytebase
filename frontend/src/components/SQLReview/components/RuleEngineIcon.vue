<template>
  <img class="w-auto h-4" :src="icon" />
</template>

<script lang="ts" setup>
import { SchemaRuleEngineType } from "@/types";
import { computed } from "vue";

const props = defineProps<{
  engine: SchemaRuleEngineType;
}>();

const icon = computed(() => {
  if (props.engine === "ORACLE") {
    return new URL("../../../assets/db-oracle.svg", import.meta.url).href;
  } else if (props.engine === "MSSQL") {
    return new URL("../../../assets/db-mssql.svg", import.meta.url).href;
  }
  return new URL(
    `../../../assets/db-${props.engine.toLowerCase()}.png`,
    import.meta.url
  ).href;
});
</script>
