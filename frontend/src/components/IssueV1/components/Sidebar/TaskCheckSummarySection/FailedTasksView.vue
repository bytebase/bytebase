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
      <div class="w-full flex flex-col mt-1 divide-y">
        <div
          class="w-full group hover:opacity-90 cursor-pointer py-2 first:pt-0"
          v-for="task in failedTasks"
          :key="task.name"
          @click="onClickTask(task)"
        >
          <p class="group-hover:underline truncate">
            {{ databaseForTask(projectOfIssue(issue), task).databaseName }}
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
import { NPopover, NTag } from "naive-ui";
import { computed } from "vue";
import { projectOfIssue, useIssueContext } from "@/components/IssueV1";
import type { PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { databaseForTask, flattenTaskV1List } from "@/utils";

const props = defineProps<{
  taskSummaryReportMap: Map<string, PlanCheckRun>;
}>();

const { issue, events } = useIssueContext();

const tasks = computed(() => flattenTaskV1List(issue.value.rolloutEntity));

const failedTasks = computed(() => {
  const tempList: Task[] = [];
  props.taskSummaryReportMap.forEach((planCheckRun, taskId) => {
    for (const result of planCheckRun.results) {
      if (result.status === Advice_Level.ERROR) {
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
    (result) => result.status === Advice_Level.ERROR
  );
  return failedResults.map((result) => result.content).join("\n");
};

const onClickTask = (task: Task) => {
  events.emit("select-task", { task });
};

defineExpose({ shouldShow });
</script>
