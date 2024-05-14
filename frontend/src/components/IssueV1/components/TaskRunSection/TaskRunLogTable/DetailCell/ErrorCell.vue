<template>
  <div class="text-sm">
    <div v-if="error" class="text-error">{{ error }}</div>
    <div v-else class="text-control-placeholder">-</div>
  </div>
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
  return undefined;
});
</script>
