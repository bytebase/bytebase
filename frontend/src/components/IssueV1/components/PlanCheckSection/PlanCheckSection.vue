<template>
  <PlanCheckRunBar
    v-if="show"
    class="px-4 py-2"
    :allow-run-checks="allowRunChecks"
    :database="database"
    :plan-name="issue.plan"
    :plan-check-run-list="planCheckRunList"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import {
  planSpecHasPlanChecks,
  specForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import PlanCheckRunBar from "@/components/PlanCheckRun/PlanCheckRunBar.vue";
import { extractUserId, useCurrentProjectV1, useCurrentUserV1 } from "@/store";
import {
  databaseForTask,
  hasProjectPermissionV2,
  isValidTaskName,
} from "@/utils";

const currentUser = useCurrentUserV1();
const { issue, selectedTask, getPlanCheckRunsForTask } = useIssueContext();
const { project } = useCurrentProjectV1();

const show = computed(() => {
  const spec = specForTask(issue.value.planEntity, selectedTask.value);
  if (!spec) {
    return false;
  }
  return planSpecHasPlanChecks(spec);
});

const database = computed(() =>
  databaseForTask(project.value, selectedTask.value)
);

const allowRunChecks = computed(() => {
  // Allowing below users to run plan checks:
  // - the creator of the issue
  // - ones who have bb.planCheckRuns.run permission in the project
  const me = currentUser.value;
  if (extractUserId(issue.value.creator) === me.email) {
    return true;
  }
  if (hasProjectPermissionV2(project.value, "bb.planCheckRuns.run")) {
    return true;
  }
  return false;
});

const planCheckRunList = computed(() => {
  // If a task is selected, show plan checks for the task.
  if (selectedTask.value && isValidTaskName(selectedTask.value.name)) {
    return getPlanCheckRunsForTask(selectedTask.value);
  }
  // Otherwise, show plan checks for the issue.
  return issue.value.planCheckRunList;
});
</script>
