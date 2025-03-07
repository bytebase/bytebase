<template>
  <NPopover v-if="shouldShow" trigger="hover" placement="bottom">
    <template #trigger>
      <NTag round type="error">
        <span class="text-red-400 text-sm mr-1">{{ $t("common.errors") }}</span>
        <span class="text-sm font-medium">
          {{ failedTasks.length }}
        </span>
      </NTag>
    </template>
    <div
      class="flex flex-col justify-start items-start w-72 max-h-128 overflow-auto"
    >
      <div class="w-full flex flex-col mt-1 gap-y-2 divide-y">
        <div
          class="w-full group hover:opacity-90 cursor-pointer"
          v-for="task in failedTasks"
          :key="task.name"
          @click="onClickTask(task)"
        >
          <p class="group-hover:underline truncate">
            {{ task.target }}
          </p>
          <p class="text-xs leading-4 text-error line-clamp-2">
            {{ failedMessageOfTask(task.name) }}
          </p>
        </div>
      </div>
    </div>
  </NPopover>
</template>

<script setup lang="ts">
import { NTag, NPopover } from "naive-ui";
import { computed } from "vue";
import { useIssueContext } from "@/components/IssueV1";
import {
  PlanCheckRun_Result_Status,
  type PlanCheckRun,
} from "@/types/proto/v1/plan_service";
import type { Task } from "@/types/proto/v1/rollout_service";
import { flattenTaskV1List } from "@/utils";

const props = defineProps<{
  taskSummaryReportMap: Map<string, PlanCheckRun>;
}>();

const { issue, events } = useIssueContext();

const tasks = computed(() => flattenTaskV1List(issue.value.rolloutEntity));

const failedTasks = computed(() => {
  const tempList: Task[] = [];
  props.taskSummaryReportMap.forEach((planCheckRun, taskId) => {
    for (const result of planCheckRun.results) {
      if (result.status === PlanCheckRun_Result_Status.ERROR) {
        tempList.push(tasks.value.find((task) => task.name === taskId) as Task);
        break;
      }
    }
  });
  return tempList;
});

const shouldShow = computed(() => failedTasks.value.length > 0);

const failedMessageOfTask = (task: string) => {
  const taskSummaryReport = props.taskSummaryReportMap.get(task);
  if (!taskSummaryReport) return "";
  const failedResults = taskSummaryReport.results.filter(
    (result) => result.status === PlanCheckRun_Result_Status.ERROR
  );
  return failedResults.map((result) => result.content).join("\n");
};

const onClickTask = (task: Task) => {
  events.emit("select-task", { task });
};

defineExpose({ shouldShow });
</script>
