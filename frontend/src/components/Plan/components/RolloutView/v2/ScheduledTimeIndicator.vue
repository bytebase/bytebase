<template>
  <span class="scheduled-time-indicator flex items-center gap-x-1 text-blue-600" :class="sizeClass">
    <ClockIcon :class="iconClass" />
    <span v-if="label">{{ label }}:</span>
    <slot>
      <span>{{ formattedTime }}</span>
    </slot>
  </span>
</template>

<script lang="ts" setup>
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { ClockIcon } from "lucide-vue-next";
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    time?: Date | Timestamp;
    label?: string;
    size?: "small" | "medium";
    format?: "time" | "datetime";
  }>(),
  {
    size: "small",
    format: "time",
  }
);

const sizeClass = computed(() => {
  return props.size === "small" ? "text-xs" : "text-sm";
});

const iconClass = computed(() => {
  return props.size === "small" ? "w-3 h-3" : "w-4 h-4";
});

const formattedTime = computed(() => {
  if (!props.time) return "";

  let date: Date;
  if (props.time instanceof Date) {
    date = props.time;
  } else {
    // Handle protobuf Timestamp
    const seconds = props.time.seconds;
    date = new Date(Number(seconds) * 1000);
  }

  if (props.format === "time") {
    return date.toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  return date.toLocaleString();
});
</script>
