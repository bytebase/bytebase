<template>
  <div class="text-sm">
    <div v-if="duration">
      {{ humanizeDurationV1(duration) }}
    </div>
    <div v-else class="text-control-placeholder">-</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { Duration } from "@/types/proto/google/protobuf/duration";
import type { FlattenLogEntry } from "./common";

const props = defineProps<{
  entry: FlattenLogEntry;
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
  const { startTime, endTime } = entry;
  if (startTime && endTime) {
    return toDuration(startTime, endTime);
  }
  return undefined;
});
</script>
