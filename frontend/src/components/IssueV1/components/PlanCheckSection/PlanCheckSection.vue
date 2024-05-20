<template>
  <div class="issue-debug">
    <h3>plan check section</h3>
    <div v-if="!isCreating">
      <NButton type="primary" @click="runPlanChecks">Run plan checks</NButton>
    </div>
    <div>
      issue.planCheckRunList.length: {{ issue.planCheckRunList.length }}
    </div>
  </div>

  <PlanCheckBar
    v-if="show"
    :allow-run-checks="allowRunChecks"
    :plan-check-run-list="planCheckRunList"
    class="px-4 py-2"
  />
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed } from "vue";
import {
  planCheckRunListForSpec,
  planCheckRunListForTask,
  planSpecHasPlanChecks,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { planServiceClient } from "@/grpcweb";
import { useCurrentUserV1 } from "@/store";
import { EMPTY_ID } from "@/types";
import { extractUserResourceName, hasProjectPermissionV2 } from "@/utils";
import PlanCheckBar from "./PlanCheckBar";

const currentUser = useCurrentUserV1();
const { isCreating, issue, selectedSpec, selectedTask } = useIssueContext();

const show = computed(() => {
  if (isCreating.value) {
    return false;
  }
  if (selectedSpec.value.id === String(EMPTY_ID)) {
    return false;
  }
  return planSpecHasPlanChecks(selectedSpec.value);
});

const allowRunChecks = computed(() => {
  // Allowing below users to run plan checks
  // - the creator of the issue
  // - the assignee of the issue
  // - ones who have bb.planCheckRuns.run permission in the project
  const me = currentUser.value;
  if (extractUserResourceName(issue.value.creator) === me.email) {
    return true;
  }
  if (extractUserResourceName(issue.value.assignee) === me.email) {
    return true;
  }
  if (
    hasProjectPermissionV2(
      issue.value.projectEntity,
      me,
      "bb.planCheckRuns.run"
    )
  ) {
    return true;
  }
  return false;
});

const planCheckRunList = computed(() => {
  // If a task is selected, show plan checks for the task.
  if (selectedTask.value && selectedTask.value.uid !== String(EMPTY_ID)) {
    return planCheckRunListForTask(issue.value, selectedTask.value);
  }
  // If a spec is selected, show plan checks for the spec.
  if (selectedSpec.value && selectedSpec.value.id !== String(EMPTY_ID)) {
    return planCheckRunListForSpec(issue.value, selectedSpec.value);
  }
  // Otherwise, show plan checks for the issue.
  return issue.value.planCheckRunList;
});

const runPlanChecks = async () => {
  const plan = issue.value.planEntity;
  if (!plan) return;

  try {
    await planServiceClient.runPlanChecks({
      name: plan.name,
    });
  } catch (ex) {
    // debugger;
  }
};
</script>
