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
    <template v-else-if="status === Task_Status.PENDING">
      <span
        v-if="active"
        class="h-2 w-2 bg-info rounded-full"
        aria-hidden="true"
      />
      <span
        v-else
        class="h-1.5 w-1.5 bg-control rounded-full"
        aria-hidden="true"
      />
    </template>
    <template v-else-if="status === Task_Status.PENDING_APPROVAL">
      <heroicons-outline:user class="w-4 h-4" />
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
    <template v-else-if="status === Task_Status.DONE">
      <SkipIcon v-if="isSkipped" class="w-5 h-5" />
      <heroicons-solid:check v-else class="w-5 h-5" />
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
import { useIssueContext } from "../logic";
import {
  PlanCheckRun_Result_Status,
  Task,
  Task_Status,
} from "@/types/proto/v1/rollout_service";

const props = defineProps<{
  active: boolean;
  status: Task_Status;
  task?: Task;
  ignorePlanCheckStatus?: boolean;
}>();

const { isCreating } = useIssueContext();

const isSkipped = computed(() => {
  // return !isCreating.value && props.task && isTaskSkipped(props.task as Task);
  return false;
});

const planCheckStatus = computed((): PlanCheckRun_Result_Status | undefined => {
  if (props.ignorePlanCheckStatus) return undefined;
  if (isCreating.value) return undefined;
  if (!props.task) return undefined;

  //     return checkStatusOfTask(task);
  return undefined; // todo
});

const classes = computed((): string => {
  if (planCheckStatus.value === PlanCheckRun_Result_Status.ERROR) {
    return "bg-white text-error !w-7";
  }
  if (planCheckStatus.value === PlanCheckRun_Result_Status.WARNING) {
    return "bg-white text-warning !w-7";
  }

  switch (props.status) {
    case Task_Status.PENDING:
      if (!isCreating.value && props.active) {
        return "bg-white border-2 border-info text-info ";
      }
      return "bg-white border-2 border-control";
    case Task_Status.PENDING_APPROVAL:
      if (!isCreating.value && props.active) {
        return "bg-white border-2 border-info text-info";
      }
      return "bg-white border-2 border-control";
    case Task_Status.RUNNING:
      return "bg-white border-2 border-info text-info";
    case Task_Status.DONE:
      if (isSkipped.value) {
        return "bg-gray-200 text-gray-500";
      }
      return "bg-success text-white";
    case Task_Status.FAILED:
      return "bg-error text-white";
    default:
      return "";
  }
});
</script>
