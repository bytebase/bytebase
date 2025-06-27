<template>
  <div class="text-sm">
    <NTooltip v-if="duration && entry.startTime && entry.endTime">
      <template #trigger>
        <span class="cursor-default">{{ humanizeDurationV1(convertDurationToNew(duration)) }}</span>
      </template>
      <template #default>
        <div>
          <p>
            Start:
            {{ dayjs(entry.startTime).format("YYYY-MM-DD HH:mm:ss.SSS UTCZZ") }}
          </p>
          <p>
            End:
            {{ dayjs(entry.endTime).format("YYYY-MM-DD HH:mm:ss.SSS UTCZZ") }}
          </p>
        </div>
      </template>
    </NTooltip>
    <div v-else class="text-control-placeholder">-</div>
  </div>
</template>

<script setup lang="ts">
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import { humanizeDurationV1 } from "@/utils";
import type { FlattenLogEntry } from "./common";
import { create } from "@bufbuild/protobuf";
import { convertDurationToNew, convertDurationToOld } from "@/utils/v1/common-conversions";

const props = defineProps<{
  entry: FlattenLogEntry;
}>();

const toDuration = (startTime: Date, endTime: Date) => {
  const ms = endTime.getTime() - startTime.getTime();
  const seconds = Math.floor(ms / 1000);
  const nanos = (ms % 1000) * 1e6;
  return convertDurationToOld(create(DurationSchema,{
    seconds: BigInt(seconds),
    nanos,
  }));
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
