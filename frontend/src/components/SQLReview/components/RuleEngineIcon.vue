<template>
  <img class="h-4 w-auto" :src="icon" />
</template>

<script lang="ts" setup>
import { SchemaRuleEngineType } from "@/types";
import { computed } from "vue";

const props = defineProps<{
  engine: SchemaRuleEngineType;
}>();

const icon = computed(() => {
  return new URL(
    `../../../assets/db-${props.engine.toLowerCase()}.png`,
    import.meta.url
  ).href;
});
</script>
