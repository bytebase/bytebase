<template>
  <div
    class="relative flex flex-shrink-0 items-center justify-center rounded-full select-none w-6 h-6 overflow-hidden"
    :class="classes"
  >
    <template v-if="planCheckStatus === PlanCheckRun_Result_Status.ERROR">
      <heroicons:exclamation-circle class="w-7 h-7 text-error" />
    </template>
    <template
      v-else-if="planCheckStatus === PlanCheckRun_Result_Status.WARNING"
    >
      <heroicons:exclamation-triangle class="w-7 h-7 text-warning" />
    </template>
    <template
      v-else-if="
        status === Task_Status.NOT_STARTED ||
        status === Task_Status.STATUS_UNSPECIFIED
      "
    >
      <heroicons-outline:user class="w-4 h-4" />
    </template>
    <template v-else-if="status === Task_Status.PENDING">
      <span class="h-1.5 w-1.5 bg-control rounded-full" aria-hidden="true" />
    </template>
    <template v-else-if="status === Task_Status.RUNNING">
      <div class="flex h-2.5 w-2.5 relative overflow-visible">
        <span
          class="w-full h-full rounded-full z-0 absolute animate-ping-slow"
          style="background-color: rgba(37, 99, 235, 0.5); /* bg-info/50 */"
          aria-hidden="true"
        />
        <span
          class="w-full h-full rounded-full z-[1] bg-info"
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
        class="w-6 h-6 rounded-full select-none bg-white border-2 border-gray-400 text-gray-400"
      />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { SkipIcon } from "@/components/Icon";
import { PlanCheckRun_Result_Status } from "@/types/proto/v1/plan_service";
import type { Task } from "@/types/proto/v1/rollout_service";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import { planCheckStatusForTask, useIssueContext } from "../logic";

const props = defineProps<{
  status: Task_Status;
  task?: Task;
  ignorePlanCheckStatus?: boolean;
}>();

const { isCreating } = useIssueContext();

const planCheckStatus = computed(() => {
  if (props.ignorePlanCheckStatus) return undefined;
  if (isCreating.value) return undefined;
  if (!props.task) return undefined;

  return planCheckStatusForTask(props.task);
});

const classes = computed((): string => {
  if (planCheckStatus.value === PlanCheckRun_Result_Status.ERROR) {
    return "bg-white text-error !w-7";
  }
  if (planCheckStatus.value === PlanCheckRun_Result_Status.WARNING) {
    return "bg-white text-warning !w-7";
  }

  switch (props.status) {
    case Task_Status.NOT_STARTED:
    case Task_Status.STATUS_UNSPECIFIED:
      if (!isCreating.value) {
        return "bg-white border-2 border-info text-info";
      }
      return "bg-white border-2 border-control";
    case Task_Status.PENDING:
      if (!isCreating.value) {
        return "bg-white border-2 border-info text-info ";
      }
      return "bg-white border-2 border-control";
    case Task_Status.RUNNING:
      return "bg-white border-2 border-info text-info";
    case Task_Status.SKIPPED:
      return "bg-gray-200 text-gray-500";
    case Task_Status.DONE:
      return "bg-success text-white";
    case Task_Status.FAILED:
      return "bg-error text-white";
    default:
      return "";
  }
});
</script>
