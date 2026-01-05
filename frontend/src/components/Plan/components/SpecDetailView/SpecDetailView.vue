<template>
  <div class="w-full flex-1 flex flex-col">
    <SpecListSection v-if="shouldShowSpecList" />
    <div
      class="w-full flex-1 flex flex-col lg:flex-row px-4 gap-4 overflow-hidden"
    >
      <!-- Left: Targets + Statement -->
      <div class="flex-1 flex flex-col min-w-0 overflow-y-auto py-3">
        <TargetListSection />
        <DataExportOptionsSection v-if="isDataExportPlan" />
        <FailedTaskRunsSection v-if="!isCreating && rollout" />
        <div class="flex-1 py-3 flex flex-col">
          <StatementSection />
        </div>
      </div>
      <!-- Right: Checks + Options -->
      <div
        v-if="shouldShowSidebar"
        class="lg:w-80 shrink-0 flex flex-col divide-y lg:border-l lg:pl-4 overflow-y-auto"
      >
        <template v-if="!specHasRelease">
          <SQLCheckV1Section v-if="isCreating" />
          <PlanCheckSection v-else />
        </template>
        <Configuration />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, type Ref } from "vue";
import { useCurrentProjectV1 } from "@/store";
import { isValidReleaseName } from "@/types";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { usePlanContext } from "../../logic/context";
import Configuration from "../Configuration";
import PlanCheckSection from "../PlanCheckSection";
import { providePlanSQLCheckContext } from "../SQLCheckSection";
import SQLCheckV1Section from "../SQLCheckV1Section";
import StatementSection from "../StatementSection";
import { useSelectedSpec } from "./context";
import DataExportOptionsSection from "./DataExportOptionsSection.vue";
import FailedTaskRunsSection from "./FailedTaskRunsSection.vue";
import SpecListSection from "./SpecListSection.vue";
import TargetListSection from "./TargetListSection.vue";

const { project } = useCurrentProjectV1();
const { isCreating, plan, issue, rollout } = usePlanContext();

const { selectedSpec } = useSelectedSpec();

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

const isCreateDatabasePlan = computed(() => {
  return plan.value.specs.every(
    (spec) => spec.config.case === "createDatabaseConfig"
  );
});

const isGrantRequestIssue = computed(() => {
  return issue.value?.type === Issue_Type.GRANT_REQUEST;
});

const shouldShowSidebar = computed(() => {
  return (
    !isDataExportPlan.value &&
    !isCreateDatabasePlan.value &&
    !isGrantRequestIssue.value
  );
});

const shouldShowSpecList = computed(() => {
  return !isDataExportPlan.value;
});
</script>
