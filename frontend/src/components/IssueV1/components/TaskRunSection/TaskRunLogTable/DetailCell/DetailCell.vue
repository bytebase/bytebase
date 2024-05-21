<template>
  <AffectedRowsCell v-if="view === 'AFFECTED_ROWS'" v-bind="props" />
  <ErrorCell v-if="view === 'ERROR'" v-bind="props" />
  <div v-if="view === 'N/A'" class="text-control-placeholder">-</div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { TaskRunLogEntry_Type } from "@/types/proto/v1/rollout_service";
import type { Sheet } from "@/types/proto/v1/sheet_service";
import type { FlattenLogEntry } from "../common";
import AffectedRowsCell from "./AffectedRowsCell.vue";
import ErrorCell from "./ErrorCell.vue";

type View = "N/A" | "ERROR" | "AFFECTED_ROWS";

const props = defineProps<{
  entry: FlattenLogEntry;
  sheet?: Sheet;
}>();

const view = computed((): View => {
  const { entry } = props;
  if (
    entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE &&
    entry.commandExecute
  ) {
    const { commandExecute } = entry;
    if (!commandExecute.raw.response) {
      return "N/A";
    }
    if (commandExecute.raw.response.error) {
      return "ERROR";
    }
    if (typeof commandExecute.affectedRows !== "undefined") {
      return "AFFECTED_ROWS";
    }
    if (typeof commandExecute.raw.response.affectedRows !== "undefined") {
      return "AFFECTED_ROWS";
    }
  }
  return "N/A";
});
</script>
