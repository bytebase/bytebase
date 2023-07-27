<template>
  <div class="issue-debug">
    <h3>plan check section</h3>
    <div v-if="!isCreating">
      <NButton type="primary" @click="runPlanChecks">Run plan checks</NButton>
    </div>
    <div>issue.planCheckRunList:</div>
    <pre>{{ issue.planCheckRunList.map(PlanCheckRun.toJSON) }}</pre>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { useIssueContext } from "../../logic";
import { rolloutServiceClient } from "@/grpcweb";
import { PlanCheckRun } from "@/types/proto/v1/rollout_service";
const { isCreating, issue } = useIssueContext();

const runPlanChecks = async () => {
  const plan = issue.value.planEntity;
  if (!plan) return;

  try {
    const response = await rolloutServiceClient.runPlanChecks({
      name: plan.name,
    });
    console.log("runPlanChecks response", response);
  } catch (ex) {
    debugger;
  }
};
</script>
