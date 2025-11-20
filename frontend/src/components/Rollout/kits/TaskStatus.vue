<template>
  <NTooltip :disabled="disabled">
    <template #trigger>
      <div
        :class="
          twMerge(
            'relative flex shrink-0 items-center justify-center rounded-full select-none overflow-hidden',
            classes
          )
        "
      >
        <template v-if="status === Task_Status.STATUS_UNSPECIFIED">
          <CircleDotDashedIcon class="w-full h-auto" />
        </template>
        <template
          v-else-if="
            status === Task_Status.NOT_STARTED
          "
        >
          <span
            class="h-1/2 w-1/2 bg-control rounded-full"
            aria-hidden="true"
          />
        </template>
        <template v-else-if="status === Task_Status.PENDING">
          <PauseIcon class="w-3/4 h-auto" />
        </template>
        <template v-else-if="status === Task_Status.RUNNING">
          <div class="flex h-1/2 w-1/2 relative overflow-visible">
            <span
              class="w-full h-full rounded-full z-0 absolute animate-ping-slow"
              style="background-color: rgba(37, 99, 235, 0.5); /* bg-info/50 */"
              aria-hidden="true"
            />
            <span
              class="w-full h-full rounded-full z-1 bg-info"
              aria-hidden="true"
            />
          </div>
        </template>
        <template v-else-if="status === Task_Status.SKIPPED">
          <FastForwardIcon class="w-3/4 h-auto" />
        </template>
        <template v-else-if="status === Task_Status.DONE">
          <heroicons-solid:check class="w-3/4 h-3/4" />
        </template>
        <template v-else-if="status === Task_Status.FAILED">
          <span
            class="rounded-full text-center font-medium text-base"
            aria-hidden="true"
            >!</span
          >
        </template>
        <template v-else-if="status === Task_Status.CANCELED">
          <heroicons-solid:minus-sm
            class="w-full h-full rounded-full select-none bg-white border-2 border-control-light text-control-light"
          />
        </template>
      </div>
    </template>
    {{ stringifyTaskStatus(status) }}
  </NTooltip>
</template>

<script lang="ts" setup>
import {
  CircleDotDashedIcon,
  FastForwardIcon,
  PauseIcon,
} from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { twMerge } from "tailwind-merge";
import { computed } from "vue";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { stringifyTaskStatus } from "@/utils";

const props = defineProps<{
  status: Task_Status;
  size?: "tiny" | "small" | "medium" | "large";
  disabled?: boolean;
}>();

const classes = computed((): string => {
  let sizeClass = "";
  switch (props.size) {
    case "tiny":
      sizeClass = "w-4 h-4";
      break;
    case "small":
      sizeClass = "w-5 h-5";
      break;
    case "medium":
      sizeClass = "w-6 h-6";
      break;
    case "large":
      sizeClass = "w-7 h-7";
      break;
    default:
      sizeClass = "w-6 h-6"; // default to medium if size is not specified
  }

  let statusClass = "";
  switch (props.status) {
    case Task_Status.NOT_STARTED:
      statusClass = "bg-white border-2 border-control";
      break;
    case Task_Status.PENDING:
      statusClass = "bg-white border-2 border-info text-info";
      break;
    case Task_Status.RUNNING:
      statusClass = "bg-white border-2 border-info text-info";
      break;
    case Task_Status.SKIPPED:
      statusClass = "bg-white border-2 border-control-light text-gray-600";
      break;
    case Task_Status.DONE:
      statusClass = "bg-success text-white";
      break;
    case Task_Status.FAILED:
      statusClass = "bg-error text-white";
      break;
    default:
      statusClass = "";
  }

  return `${sizeClass} ${statusClass}`;
});
</script>
