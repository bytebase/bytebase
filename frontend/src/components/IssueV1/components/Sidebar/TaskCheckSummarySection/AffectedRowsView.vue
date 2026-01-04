<template>
  <NPopover v-if="shouldShow" trigger="hover" placement="bottom">
    <template #trigger>
      <NTag round>
        <span class="text-gray-400 text-sm mr-1">{{
          $t("task.check-type.affected-rows.self")
        }}</span>
        <span class="text-sm font-medium">
          {{ affectedRows }}
        </span>
      </NTag>
    </template>
    <div
      class="flex flex-col justify-start items-start w-72 max-h-128 overflow-auto"
    >
      <p class="text-sm text-gray-400">
        {{ $t("task.check-type.affected-rows.description") }}
      </p>
      <div class="w-full flex flex-col mt-1 gap-y-1">
        <div
          class="w-full flex flex-row justify-between items-center gap-1 group hover:opacity-90 cursor-pointer"
          v-for="item in affectedTasks"
          :key="item.task.name"
          @click="onClickTask(item.task)"
        >
          <span class="group-hover:underline truncate">
            {{ databaseForTask(projectOfIssue(issue), item.task).databaseName }}
          </span>
          <span
            class="shrink-0 opacity-80"
          >
            {{ item.count }}
          </span>
        </div>
      </div>
    </div>
  </NPopover>
</template>

<script setup lang="ts">
import { NPopover, NTag } from "naive-ui";
import { computed } from "vue";
import { projectOfIssue, useIssueContext } from "@/components/IssueV1/logic";
import { type PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { databaseForTask, flattenTaskV1List } from "@/utils";

const props = defineProps<{
  taskSummaryReportMap: Map<string, PlanCheckRun>;
}>();

const { issue, events } = useIssueContext();

const tasks = computed(() => flattenTaskV1List(issue.value.rolloutEntity));

const affectedTasks = computed(() => {
  const tempMap = new Map<string, { task: Task; count: bigint }>();

  for (const [taskName, planCheckRun] of props.taskSummaryReportMap.entries()) {
    if (
      planCheckRun.results.every(
        (result) =>
          result.report?.case !== "sqlSummaryReport" ||
          result.report.value.affectedRows === undefined
      )
    ) {
      continue;
    }

    const task = tasks.value.find((t) => t.name === taskName);
    if (!task) {
      continue;
    }

    const count = planCheckRun.results.reduce((acc, result) => {
      if (result.report?.case === "sqlSummaryReport") {
        return acc + (result.report.value.affectedRows || 0n);
      }
      return acc;
    }, 0n);

    tempMap.set(taskName, { task, count });
  }
  return Array.from(tempMap.values()).sort((a, b) => Number(b.count - a.count));
});

const summaryReportResults = computed(() =>
  Array.from(props.taskSummaryReportMap.values()).flatMap(
    (report) => report.results
  )
);

const affectedRows = computed(() =>
  summaryReportResults.value.reduce((acc, result) => {
    if (result.report?.case === "sqlSummaryReport") {
      return acc + (result.report.value.affectedRows || 0n);
    }
    return acc;
  }, 0n)
);

const shouldShow = computed(() =>
  summaryReportResults.value.some(
    (result) =>
      result.report?.case === "sqlSummaryReport" &&
      result.report.value.affectedRows !== undefined
  )
);

const onClickTask = (task: Task) => {
  events.emit("select-task", { task });
};

defineExpose({ shouldShow });
</script>
