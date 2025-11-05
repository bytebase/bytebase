<template>
  <NTooltip :disabled="!showTooltip">
    <template #trigger>
      <div
        class="relative flex shrink-0 items-center justify-center select-none overflow-hidden"
        :class="[classes, containerSize]"
      >
        <template
          v-if="
            status === Task_Status.NOT_STARTED ||
            status === Task_Status.STATUS_UNSPECIFIED
          "
        >
          <heroicons-outline:user class="w-4 h-4" />
        </template>
        <template v-else-if="status === Task_Status.PENDING">
          <span
            class="h-1.5 w-1.5 bg-control rounded-full"
            aria-hidden="true"
          />
        </template>
        <template v-else-if="status === Task_Status.RUNNING">
          <div class="flex h-2.5 w-2.5 relative overflow-visible">
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
          <SkipIcon class="w-5 h-5" />
        </template>
        <template v-else-if="status === Task_Status.DONE">
          <heroicons-solid:check class="w-5 h-5" />
        </template>
        <template v-else-if="status === Task_Status.FAILED">
          <span
            class="h-2.5 w-2.5 rounded-full text-center pb-6 font-medium text-base"
            aria-hidden="true"
            >!</span
          >
        </template>
        <template v-else-if="status === Task_Status.CANCELED">
          <heroicons-solid:minus-sm
            class="rounded-full select-none bg-white border-2 border-gray-400 text-gray-400"
            :class="containerSize"
          />
        </template>
      </div>
    </template>
    {{ Task_Status[status] }}
  </NTooltip>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { SkipIcon } from "@/components/Icon";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { useIssueContext } from "../logic";

const props = withDefaults(
  defineProps<{
    status: Task_Status;
    size?: "small" | "medium" | "large";
    showTooltip?: boolean;
  }>(),
  {
    size: "medium",
    showTooltip: true,
  }
);

const { isCreating } = useIssueContext();

const containerSize = computed((): string => {
  switch (props.size) {
    case "small":
      return "w-4 h-4";
    case "medium":
      return "w-6 h-6";
    case "large":
      return "w-8 h-8";
    default:
      return "w-6 h-6";
  }
});

const classes = computed((): string => {
  const borderSize = props.size === "small" ? "border" : "border-2";
  switch (props.status) {
    case Task_Status.NOT_STARTED:
    case Task_Status.STATUS_UNSPECIFIED:
      if (!isCreating.value) {
        return `bg-white border-info text-info rounded-full ${borderSize}`;
      }
      return `bg-white border-control rounded-full ${borderSize}`;
    case Task_Status.PENDING:
      if (!isCreating.value) {
        return `bg-white border-info text-info rounded-full ${borderSize}`;
      }
      return `bg-white border-control rounded-full ${borderSize}`;
    case Task_Status.RUNNING:
      return `bg-white border-info text-info rounded-full ${borderSize}`;
    case Task_Status.SKIPPED:
      return "bg-gray-200 text-gray-500 rounded-full";
    case Task_Status.DONE:
      return "bg-success text-white rounded-full";
    case Task_Status.FAILED:
      return "bg-error text-white rounded-full";
    default:
      return "";
  }
});
</script>
