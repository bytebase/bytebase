<template>
  <div class="text-sm">
    <div v-if="duration">
      {{ humanizeDurationV1(duration) }}
    </div>
    <div v-else class="text-control-placeholder">N/A</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { Duration } from "@/types/proto/google/protobuf/duration";
import {
  TaskRunLogEntry_Type,
  type TaskRunLogEntry,
} from "@/types/proto/v1/rollout_service";

const props = defineProps<{
  entry: TaskRunLogEntry;
}>();

const toDuration = (startTime: Date, endTime: Date) => {
  const ms = endTime.getTime() - startTime.getTime();
  const seconds = Math.floor(ms / 1000);
  const nanos = (ms % 1000) * 1e6;
  return Duration.fromPartial({
    seconds,
    nanos,
  });
};

const duration = computed(() => {
  const { entry } = props;
  if (entry.type === TaskRunLogEntry_Type.SCHEMA_DUMP && entry.schemaDump) {
    const { startTime, endTime } = entry.schemaDump;
    if (startTime && endTime) {
      return toDuration(startTime, endTime);
    }
  }
  if (
    entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE &&
    entry.commandExecute
  ) {
    const exec = entry.commandExecute;
    const startTime = exec.logTime;
    const endTime = exec.response?.logTime;
    if (startTime && endTime) {
      return toDuration(startTime, endTime);
    }
  }
  return undefined;
});
</script>
