<template>
  <NPerformantEllipsis
    class="w-full cursor-pointer leading-6 hover:text-accent"
  >
    <span @click="$emit('click')">
      {{ table.name }}
    </span>
  </NPerformantEllipsis>
</template>

<script lang="ts" setup>
import { NPerformantEllipsis } from "naive-ui";
import { Table } from "@/types/v1/schemaEditor";

defineProps<{
  table: Table;
}>();
defineEmits<{
  (event: "click"): void;
}>();
</script>
