<template>
  <div class="text-sm">
    <div v-if="!isNaN(affectedRows)">{{ affectedRows }}</div>
    <div v-else class="text-control-placeholder">-</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { TaskRunLogEntry_Type } from "@/types/proto/v1/rollout_service";
import type { Sheet } from "@/types/proto/v1/sheet_service";
import type { FlattenLogEntry } from "./common";

const props = defineProps<{
  entry: FlattenLogEntry;
  sheet?: Sheet;
}>();

const affectedRows = computed(() => {
  const { entry } = props;
  if (entry.type === TaskRunLogEntry_Type.SCHEMA_DUMP && entry.schemaDump) {
    return Number.NaN;
  }
  if (
    entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE &&
    entry.commandExecute
  ) {
    const exec = entry.commandExecute;
    return exec.affectedRows ?? Number.NaN;
  }
  return Number.NaN;
});
</script>
