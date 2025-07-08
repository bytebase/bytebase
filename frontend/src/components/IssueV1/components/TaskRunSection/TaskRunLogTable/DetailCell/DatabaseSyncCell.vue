<template>
  <span v-if="error" class="text-error">{{ error }}</span>
  <span v-else class="text-control-placeholder">-</span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { TaskRunLogEntry_Type } from "@/types/proto-es/v1/rollout_service_pb";
import type { FlattenLogEntry } from "../common";

const props = defineProps<{
  entry: FlattenLogEntry;
}>();

const error = computed(() => {
  const { type, databaseSync } = props.entry;
  if (type === TaskRunLogEntry_Type.DATABASE_SYNC && databaseSync) {
    return databaseSync.error;
  }
  return "";
});
</script>
