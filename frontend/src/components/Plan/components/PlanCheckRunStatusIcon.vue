<template>
  <span
    v-if="hasAnyChecks"
    class="flex items-center justify-center select-none shrink-0"
    :class="iconClass()"
  >
    <template v-if="planCheckRunStatus === Advice_Level.ERROR">
      <XIcon />
    </template>
    <template v-else-if="planCheckRunStatus === Advice_Level.WARNING">
      <span class="text-xs font-bold" aria-hidden="true">!</span>
    </template>
    <template v-else-if="planCheckRunStatus === Advice_Level.SUCCESS">
      <CheckIcon />
    </template>
    <template v-else-if="hasRunningChecks">
      <span
        class="w-2.5 h-2.5 rounded-full border-2 border-control"
        aria-hidden="true"
      ></span>
    </template>
  </span>
</template>

<script lang="ts" setup>
import type { PropType } from "vue";
import { computed } from "vue";
import type { Plan, PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import CheckIcon from "~icons/heroicons-solid/check";
import XIcon from "~icons/heroicons-solid/x";
import { useResolvedPlanCheckStatus } from "../logic";

export type SizeType = "small" | "normal";

const props = defineProps({
  plan: {
    required: true,
    type: Object as PropType<Plan>,
  },
  planCheckRuns: {
    type: Array as PropType<PlanCheckRun[] | undefined>,
    default: undefined,
  },
  size: {
    type: String as PropType<SizeType>,
    default: "normal",
  },
});

const planCheckRunsRef = computed(() => props.planCheckRuns);

const {
  overallAdviceLevel: planCheckRunStatus,
  hasAnyStatus: hasAnyChecks,
  hasRunning: hasRunningChecks,
} = useResolvedPlanCheckStatus(
  computed(() => props.plan),
  planCheckRunsRef
);

const iconClass = () => {
  const sizeClass = props.size === "normal" ? "w-4 h-4" : "w-3.5 h-3.5";
  switch (planCheckRunStatus.value) {
    case Advice_Level.ERROR:
      return `${sizeClass} text-error`;
    case Advice_Level.WARNING:
      return `${sizeClass} text-warning`;
    case Advice_Level.SUCCESS:
      return `${sizeClass} text-success`;
    default:
      return `${sizeClass} text-warning`;
  }
};
</script>
