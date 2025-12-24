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
import { create } from "@bufbuild/protobuf";
import { computed, ref } from "vue";
import { useIssueContext } from "@/components/IssueV1";
import {
  type PlanCheckRun,
  type PlanCheckRun_Result,
  PlanCheckRun_Result_Type,
  PlanCheckRunSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { flattenTaskV1List, isDatabaseChangeRelatedIssue } from "@/utils";
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

// With consolidated model, create synthetic PlanCheckRuns per task
// containing only the summary report results for that task's target
const taskSummaryReportMap = computed(() => {
  const { planCheckRunList } = issue.value;
  const tempMap = new Map<string, PlanCheckRun>();

  for (const task of tasks.value) {
    // Find all summary report results that match this task's target
    const taskResults: PlanCheckRun_Result[] = [];

    for (const run of planCheckRunList) {
      const matchingResults = run.results.filter(
        (result) =>
          result.type === PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT &&
          result.target === task.target
      );
      taskResults.push(...matchingResults);
    }

    if (taskResults.length > 0) {
      // Create a synthetic PlanCheckRun with just the results for this task
      tempMap.set(
        task.name,
        create(PlanCheckRunSchema, {
          results: taskResults,
        })
      );
    }
  }

  return tempMap;
});
</script>
