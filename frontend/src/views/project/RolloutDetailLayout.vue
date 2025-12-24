<template>
  <router-view />
</template>

<script lang="ts" setup>
import { watchEffect } from "vue";
import { usePlanContextWithRollout } from "@/components/Plan";
import { provideRolloutViewContext } from "@/components/RolloutV1/logic/context";
import { usePolicyV1Store } from "@/store";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";

const { rollout } = usePlanContextWithRollout();
const policyStore = usePolicyV1Store();

provideRolloutViewContext();

// Prepare rollout policies for all stages in the rollout.
watchEffect(() => {
  if (!rollout.value) return;

  const environmentNames = new Set<string>();
  // Collect all unique environment names from stages
  for (const stage of rollout.value.stages) {
    if (stage.environment) {
      environmentNames.add(stage.environment);
    }
  }

  // Fetch rollout policies for each environment
  for (const environmentName of environmentNames) {
    policyStore
      .getOrFetchPolicyByParentAndType({
        parentPath: environmentName,
        policyType: PolicyType.ROLLOUT_POLICY,
      })
      .catch(() => {
        // Silently ignore errors as policies might not exist
      });
  }
});
</script>
