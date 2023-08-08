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
    v-if="!isCreating"
    :allow-run-checks="allowRunChecks"
    :task="selectedTask"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { NButton } from "naive-ui";

import { rolloutServiceClient } from "@/grpcweb";
import { useCurrentUserV1 } from "@/store";
import {
  extractUserResourceName,
  hasWorkspacePermissionV1,
  isOwnerOfProjectV1,
} from "@/utils";
import { useIssueContext } from "@/components/IssueV1/logic";
import PlanCheckBar from "./PlanCheckBar";

const currentUser = useCurrentUserV1();
const { isCreating, issue, selectedTask } = useIssueContext();

const allowRunChecks = computed(() => {
  // Allowing below users to run plan checks
  // - the creator of the issue
  // - the assignee of the issue
  // - project owners
  // - workspace DBAs/owners
  const me = currentUser.value;
  if (extractUserResourceName(issue.value.creator) === me.email) {
    return true;
  }
  if (extractUserResourceName(issue.value.assignee) === me.email) {
    return true;
  }
  if (isOwnerOfProjectV1(issue.value.projectEntity.iamPolicy, me)) {
    return true;
  }
  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-issue",
      me.userRole
    )
  ) {
    return true;
  }
  return false;
});

const runPlanChecks = async () => {
  const plan = issue.value.planEntity;
  if (!plan) return;

  try {
    const response = await rolloutServiceClient.runPlanChecks({
      name: plan.name,
    });
    console.log("runPlanChecks response", response);
  } catch (ex) {
    // debugger;
  }
};
</script>
