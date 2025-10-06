<template>
  <span
    v-if="hasAnyChecks"
    class="flex items-center justify-center select-none flex-shrink-0"
    :class="iconClass()"
  >
    <template v-if="planCheckRunStatus === PlanCheckRun_Result_Status.ERROR">
      <XIcon />
    </template>
    <template
      v-else-if="planCheckRunStatus === PlanCheckRun_Result_Status.WARNING"
    >
      <span class="text-xs font-bold" aria-hidden="true">!</span>
    </template>
    <template
      v-else-if="planCheckRunStatus === PlanCheckRun_Result_Status.SUCCESS"
    >
      <CheckIcon />
    </template>
    <template v-else-if="hasRunningChecks">
      <span
        class="w-1.5 h-1.5 rounded-full bg-warning animate-pulse"
        aria-hidden="true"
      ></span>
    </template>
  </span>
</template>

<script lang="ts" setup>
import type { PropType } from "vue";
import { computed } from "vue";
import CheckIcon from "~icons/heroicons-solid/check";
import XIcon from "~icons/heroicons-solid/x";
import {
  PlanCheckRun_Result_Status,
  type Plan,
} from "@/types/proto-es/v1/plan_service_pb";
import { usePlanCheckStatus } from "../logic";

export type SizeType = "small" | "normal";

const props = defineProps({
  plan: {
    required: true,
    type: Object as PropType<Plan>,
  },
  size: {
    type: String as PropType<SizeType>,
    default: "normal",
  },
});

const {
  getOverallStatus: planCheckRunStatus,
  hasAnyStatus: hasAnyChecks,
  hasRunning: hasRunningChecks,
} = usePlanCheckStatus(computed(() => props.plan));

const iconClass = () => {
  const sizeClass = props.size === "normal" ? "w-4 h-4" : "w-3.5 h-3.5";
  switch (planCheckRunStatus.value) {
    case PlanCheckRun_Result_Status.ERROR:
      return `${sizeClass} text-error`;
    case PlanCheckRun_Result_Status.WARNING:
      return `${sizeClass} text-warning`;
    case PlanCheckRun_Result_Status.SUCCESS:
      return `${sizeClass} text-success`;
    default:
      return `${sizeClass} text-warning`;
  }
};
</script>
