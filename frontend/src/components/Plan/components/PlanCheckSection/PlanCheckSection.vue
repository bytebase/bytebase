<template>
  <PlanCheckRunBar
    v-if="show"
    class="px-4 py-2"
    :allow-run-checks="allowRunChecks"
    :database="database"
    :plan-name="plan.name"
    :plan-check-run-list="planCheckRunList"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import {
  isDatabaseChangeSpec,
  planCheckRunListForSpec,
  planSpecHasPlanChecks,
  usePlanContext,
  databaseForSpec,
} from "@/components/Plan/logic";
import PlanCheckRunBar from "@/components/PlanCheckRun/PlanCheckRunBar.vue";
import { useCurrentUserV1, extractUserId } from "@/store";
import { EMPTY_ID } from "@/types";
import { hasProjectPermissionV2 } from "@/utils";

const currentUser = useCurrentUserV1();
const { plan, selectedSpec } = usePlanContext();

const show = computed(() => {
  if (selectedSpec.value.id === String(EMPTY_ID)) {
    return false;
  }
  return planSpecHasPlanChecks(selectedSpec.value);
});

const database = computed(() =>
  databaseForSpec(plan.value, selectedSpec.value)
);

const allowRunChecks = computed(() => {
  // Allowing below users to run plan checks
  // - the creator of the plan
  // - ones who have bb.planCheckRuns.run permission in the project
  const me = currentUser.value;
  if (extractUserId(plan.value.creator) === me.email) {
    return true;
  }
  if (
    hasProjectPermissionV2(plan.value.projectEntity, "bb.planCheckRuns.run")
  ) {
    return true;
  }
  return false;
});

const planCheckRunList = computed(() => {
  // If a spec is database change spec, show plan checks for the spec.
  if (selectedSpec.value && isDatabaseChangeSpec(selectedSpec.value)) {
    return planCheckRunListForSpec(plan.value, selectedSpec.value);
  }
  // Otherwise, show all plan checks in the plan.
  return plan.value.planCheckRunList;
});
</script>
