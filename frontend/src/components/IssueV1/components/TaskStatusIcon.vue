<template>
  <div
    class="relative flex shrink-0 items-center justify-center select-none w-6 h-6 overflow-hidden"
    :class="classes"
  >
    <template v-if="checkStatus === Advice_Level.ERROR">
      <CircleAlertIcon class="text-error" />
    </template>
    <template v-else-if="checkStatus === Advice_Level.WARNING">
      <TriangleAlertIcon class="text-warning" />
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
        class="w-6 h-6 rounded-full select-none bg-white border-2 border-gray-400 text-gray-400"
      />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { CircleAlertIcon, TriangleAlertIcon } from "lucide-vue-next";
import { computed } from "vue";
import { SkipIcon } from "@/components/Icon";
import { usePlanSQLCheckContext } from "@/components/Plan/components/SQLCheckSection/context";
import { useCurrentProjectV1 } from "@/store";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { databaseForTask } from "@/utils";
import { planCheckStatusForTask, useIssueContext } from "../logic";

const props = defineProps<{
  status: Task_Status;
  task?: Task;
  ignoreCheckStatus?: boolean;
}>();

const { isCreating } = useIssueContext();
const { project } = useCurrentProjectV1();
const { resultMap } = usePlanSQLCheckContext();

const checkStatus = computed(() => {
  if (props.ignoreCheckStatus) return undefined;
  if (!props.task) return undefined;
  if (isCreating.value) {
    const checkResult =
      resultMap.value[databaseForTask(project.value, props.task).name];
    if (!checkResult) return undefined;

    if (
      checkResult.advices.some((advice) => advice.status === Advice_Level.ERROR)
    ) {
      return Advice_Level.ERROR;
    } else if (
      checkResult.advices.some(
        (advice) => advice.status === Advice_Level.WARNING
      )
    ) {
      return Advice_Level.WARNING;
    }
    return undefined;
  }
  return planCheckStatusForTask(props.task);
});

const classes = computed((): string => {
  if (Boolean(checkStatus.value)) {
    return "";
  }

  switch (props.status) {
    case Task_Status.NOT_STARTED:
    case Task_Status.STATUS_UNSPECIFIED:
      if (!isCreating.value) {
        return "bg-white border-2 border-info text-info rounded-full";
      }
      return "bg-white border-2 border-control rounded-full";
    case Task_Status.PENDING:
      if (!isCreating.value) {
        return "bg-white border-2 border-info text-info rounded-full";
      }
      return "bg-white border-2 border-control rounded-full";
    case Task_Status.RUNNING:
      return "bg-white border-2 border-info text-info rounded-full";
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
