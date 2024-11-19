<template>
  <span v-if="error" class="text-error">{{ error }}</span>
  <span v-else class="text-control-placeholder">-</span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { TaskRunLogEntry_Type } from "@/types/proto/v1/rollout_service";
import type { FlattenLogEntry } from "../common";

const props = defineProps<{
  entry: FlattenLogEntry;
}>();

const error = computed(() => {
  const { entry } = props;
  if (
    entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE &&
    entry.commandExecute
  ) {
    const exec = entry.commandExecute;
    return exec.raw.response?.error;
  }
  if (entry.type === TaskRunLogEntry_Type.PRIOR_BACKUP && entry.priorBackup) {
    return entry.priorBackup.error;
  }
  return undefined;
});
</script>
