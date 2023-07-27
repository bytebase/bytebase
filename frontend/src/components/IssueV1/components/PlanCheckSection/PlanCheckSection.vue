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

  <PlanCheckBar v-if="!isCreating" />
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";

import { rolloutServiceClient } from "@/grpcweb";
import { useIssueContext } from "../../logic";

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
