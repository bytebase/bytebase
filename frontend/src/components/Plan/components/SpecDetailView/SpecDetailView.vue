<template>
  <div class="w-full flex-1 flex flex-col">
    <SpecListSection v-if="shouldShowSpecList" />
    <div
      class="w-full flex-1 flex flex-col gap-3 px-4 divide-y overflow-x-auto"
    >
      <TargetListSection />
      <DataExportOptionsSection v-if="isDataExportPlan" />
      <FailedTaskRunsSection v-if="!isCreating && rollout" />
      <template v-if="!specHasRelease">
        <SQLCheckV1Section v-if="isCreating" />
        <PlanCheckSection v-else />
      </template>
      <div class="w-full flex-1 space-y-3 pt-3 flex flex-col">
        <StatementSection />
        <Configuration />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, type Ref } from "vue";
import { useCurrentProjectV1 } from "@/store";
import { isValidReleaseName } from "@/types";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { usePlanContext } from "../../logic/context";
import Configuration from "../Configuration";
import PlanCheckSection from "../PlanCheckSection";
import { providePlanSQLCheckContext } from "../SQLCheckSection";
import SQLCheckV1Section from "../SQLCheckV1Section";
import StatementSection from "../StatementSection";
import DataExportOptionsSection from "./DataExportOptionsSection.vue";
import FailedTaskRunsSection from "./FailedTaskRunsSection.vue";
import SpecListSection from "./SpecListSection.vue";
import TargetListSection from "./TargetListSection.vue";
import { useSelectedSpec } from "./context";

const { project } = useCurrentProjectV1();
const { isCreating, plan, rollout } = usePlanContext();

const selectedSpec = useSelectedSpec();

providePlanSQLCheckContext({
  project,
  plan,
  selectedSpec: selectedSpec as Ref<Plan_Spec>,
});

const specHasRelease = computed(() => {
  return (
    selectedSpec.value.config.case === "changeDatabaseConfig" &&
    isValidReleaseName(selectedSpec.value.config.value.release)
  );
});

const isDataExportPlan = computed(() => {
  return plan.value.specs.every(
    (spec) => spec.config.case === "exportDataConfig"
  );
});

const shouldShowSpecList = computed(() => {
  return (
    (isCreating.value ||
      plan.value.specs.length > 1 ||
      plan.value.rollout === "") &&
    !isDataExportPlan.value
  );
});
</script>
