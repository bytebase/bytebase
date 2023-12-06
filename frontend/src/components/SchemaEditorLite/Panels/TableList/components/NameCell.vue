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
import { TableMetadata } from "@/types/proto/v1/database_service";

defineProps<{
  table: TableMetadata;
}>();
defineEmits<{
  (event: "click"): void;
}>();
</script>
