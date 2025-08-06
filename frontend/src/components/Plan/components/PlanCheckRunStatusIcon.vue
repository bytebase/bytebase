<template>
  <span
    class="flex items-center justify-center rounded-full select-none overflow-hidden"
    :class="iconClass()"
  >
    <template v-if="planCheckRunStatus === PlanCheckRun_Result_Status.ERROR">
      <span
        class="h-2 w-2 rounded-full text-center pb-6 font-normal text-base"
        aria-hidden="true"
        >!</span
      >
    </template>
    <template
      v-else-if="planCheckRunStatus === PlanCheckRun_Result_Status.WARNING"
    >
      <span
        class="h-2 w-2 rounded-full text-center pb-6 font-normal text-base"
        aria-hidden="true"
        >!</span
      >
    </template>
    <template
      v-else-if="planCheckRunStatus === PlanCheckRun_Result_Status.SUCCESS"
    >
      <heroicons-solid:check class="w-4 h-4" />
    </template>
    <template v-else>
      <span class="h-3 w-3 bg-white rounded-full" aria-hidden="true"></span>
    </template>
  </span>
</template>

<script lang="ts" setup>
import type { PropType } from "vue";
import { computed } from "vue";
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

const { getOverallStatus: planCheckRunStatus } = usePlanCheckStatus(
  computed(() => props.plan)
);

const iconClass = () => {
  const iconClass = props.size === "normal" ? "w-5 h-5" : "w-4 h-4";
  switch (planCheckRunStatus.value) {
    case PlanCheckRun_Result_Status.ERROR:
      return iconClass + " bg-error text-white";
    case PlanCheckRun_Result_Status.WARNING:
      return iconClass + " bg-warning text-white";
    case PlanCheckRun_Result_Status.SUCCESS:
      return iconClass + " bg-success text-white";
    default:
      return iconClass + " bg-control text-white";
  }
};
</script>
