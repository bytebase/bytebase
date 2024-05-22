<template>
  <PlanCheckBar
    v-if="show"
    :allow-run-checks="allowRunChecks"
    :plan-check-run-list="planCheckRunList"
    class="px-4 py-2"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import {
  planCheckRunListForSpec,
  planSpecHasPlanChecks,
  usePlanContext,
} from "@/components/Plan/logic";
import { useCurrentUserV1 } from "@/store";
import { EMPTY_ID } from "@/types";
import { extractUserResourceName, hasProjectPermissionV2 } from "@/utils";
import PlanCheckBar from "./PlanCheckBar";

const currentUser = useCurrentUserV1();
const { isCreating, plan, selectedSpec } = usePlanContext();

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
  // - the creator of the plan
  // - ones who have bb.planCheckRuns.run permission in the project
  const me = currentUser.value;
  if (extractUserResourceName(plan.value.creator) === me.email) {
    return true;
  }
  if (
    hasProjectPermissionV2(plan.value.projectEntity, me, "bb.planCheckRuns.run")
  ) {
    return true;
  }
  return false;
});

const planCheckRunList = computed(() => {
  // If a spec is selected, show plan checks for the spec.
  if (selectedSpec.value && selectedSpec.value.id !== String(EMPTY_ID)) {
    return planCheckRunListForSpec(plan.value, selectedSpec.value);
  }
  // Otherwise, show plan checks for the plan.
  return plan.value.planCheckRunList;
});
</script>
