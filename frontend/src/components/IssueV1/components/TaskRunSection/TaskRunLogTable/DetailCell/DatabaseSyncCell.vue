<template>
  <div class="text-sm">
    <div v-if="error" class="text-error">{{ error }}</div>
    <div v-else class="text-control-placeholder">-</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { TaskRunLogEntry_Type } from "@/types/proto/v1/rollout_service";
import type { Sheet } from "@/types/proto/v1/sheet_service";
import type { FlattenLogEntry } from "../common";

const props = defineProps<{
  entry: FlattenLogEntry;
  sheet?: Sheet;
}>();

const error = computed(() => {
  const { type, databaseSync } = props.entry;
  if (type === TaskRunLogEntry_Type.DATABASE_SYNC && databaseSync) {
    return databaseSync.error;
  }
  return "";
});
</script>
