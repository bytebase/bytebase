<template>
  <div
    class="flex-1 flex flex-col hide-scrollbar gap-3 divide-y overflow-x-hidden"
  >
    <div class="w-full flex flex-col">
      <SpecListSection v-if="shouldShowSpecList" />
      <TargetListSection />
      <FailedTaskRunsSection v-if="!isCreating && rollout" />
    </div>
    <template v-if="!specHasRelease">
      <SQLCheckV1Section v-if="isCreating" />
      <PlanCheckSection v-else />
    </template>
    <div class="w-full space-y-3 pt-3">
      <StatementSection />
      <Configuration />
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

const shouldShowSpecList = computed(() => {
  return (
    isCreating.value || plan.value.specs.length > 1 || plan.value.rollout === ""
  );
});
</script>
