<template>
  <div v-show="shouldShow" class="flex flex-col gap-y-2">
    <div class="flex items-center gap-x-1 textlabel">
      <span>{{ $t("common.summary") }}</span>
    </div>
    <div
      class="w-full flex flex-row justify-start items-center gap-1 flex-wrap"
    >
      <AffectedRowsView
        ref="affectedRowsViewRef"
        :task-summary-report-map="taskSummaryReportMap"
      />
      <FailedTasksView
        ref="failedTasksViewRef"
        :task-summary-report-map="taskSummaryReportMap"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useIssueContext } from "@/components/IssueV1";
import {
  PlanCheckRun_Type,
  type PlanCheckRun,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  flattenTaskV1List,
  isDatabaseChangeRelatedIssue,
  sheetNameOfTaskV1,
} from "@/utils";
import AffectedRowsView from "./AffectedRowsView.vue";
import FailedTasksView from "./FailedTasksView.vue";

const { issue, isCreating } = useIssueContext();

const affectedRowsViewRef = ref<InstanceType<typeof AffectedRowsView>>();
const failedTasksViewRef = ref<InstanceType<typeof FailedTasksView>>();

const shouldShow = computed(() => {
  if (isCreating.value) return false;
  if (!isDatabaseChangeRelatedIssue(issue.value)) {
    return false;
  }
  if (
    !affectedRowsViewRef.value?.shouldShow &&
    !failedTasksViewRef.value?.shouldShow
  ) {
    return false;
  }
  return true;
});

const tasks = computed(() => flattenTaskV1List(issue.value.rolloutEntity));

const taskSummaryReportMap = computed(() => {
  const { planCheckRunList } = issue.value;
  const summaryReports = planCheckRunList.filter(
    (report) =>
      report.type === PlanCheckRun_Type.DATABASE_STATEMENT_SUMMARY_REPORT
  );
  const tempMap = new Map<string, PlanCheckRun>();
  for (const task of tasks.value) {
    const summaryReport = summaryReports.find(
      (report) =>
        report.target === task.target &&
        report.sheet === sheetNameOfTaskV1(task)
    );
    if (summaryReport) {
      tempMap.set(task.name, summaryReport);
    }
  }
  return tempMap;
});
</script>
