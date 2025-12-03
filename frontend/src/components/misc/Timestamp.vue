<template>
  <NTooltip v-if="hasValidTimestamp" :disabled="!showTooltip" class="text-sm">
    <template #trigger>
      <span class="text-control-light" :class="customClass">
        {{ humanizedTime }}
      </span>
    </template>
    <template #default>
      <div class="whitespace-nowrap">
        {{ fullDateTime }}
      </div>
    </template>
  </NTooltip>
  <span v-else class="text-control-light" :class="customClass">
    {{ fallbackText }}
  </span>
</template>

<script setup lang="ts">
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import timezone from "dayjs/plugin/timezone";
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { getTimeForPbTimestampProtoEs } from "@/types/timestamp";
import { humanizeTs } from "@/utils/util";

dayjs.extend(timezone);

interface Props {
  timestamp?: Timestamp;
  fallbackText?: string;
  showTooltip?: boolean;
  customClass?: string;
}

const props = withDefaults(defineProps<Props>(), {
  fallbackText: "-",
  showTooltip: true,
  customClass: "",
});

// Check if we have a valid timestamp
const hasValidTimestamp = computed(() => {
  return props.timestamp && (props.timestamp.seconds || props.timestamp.nanos);
});

// Get timestamp in milliseconds
const timestampInMilliseconds = computed(() => {
  if (!hasValidTimestamp.value) return null;
  return getTimeForPbTimestampProtoEs(props.timestamp, 0);
});

// Convert to seconds for humanizeTs
const timestampInSeconds = computed(() => {
  if (!timestampInMilliseconds.value) return null;
  return Math.floor(timestampInMilliseconds.value / 1000);
});

// Humanized time using the existing humanizeTs function
const humanizedTime = computed(() => {
  if (!timestampInSeconds.value) return props.fallbackText;
  return humanizeTs(timestampInSeconds.value);
});

const fullDateTime = computed(() => {
  if (!timestampInMilliseconds.value) return "";

  const date = dayjs(timestampInMilliseconds.value);
  return date.local().format();
});
</script>
