<template>
  <span v-if="!isNaN(affectedRows)">
    {{
      $t("issue.task-run.task-run-log.affected-rows-n", {
        n: affectedRows,
      })
    }}
  </span>
  <span v-else class="text-control-placeholder">-</span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { TaskRunLogEntry_Type } from "@/types/proto-es/v1/rollout_service_pb";
import type { FlattenLogEntry } from "../common";

const props = defineProps<{
  entry: FlattenLogEntry;
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
