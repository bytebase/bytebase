<template>
  <div class="w-full flex-1 flex flex-col">
    <SpecListSection />
    <div class="w-full flex-1 flex flex-col px-4 gap-y-4 py-4 overflow-y-auto">
      <TargetListSection />

      <!-- Statement -->
      <div :class="[plan.hasRollout ? 'h-[192px]' : 'h-[256px]', 'flex flex-col']">
        <StatementSection />
      </div>

      <!-- Checks + Configuration -->
      <template v-if="!specHasRelease">
        <SQLCheckV1Section v-if="isCreating" />
        <PlanCheckSection v-else />
      </template>
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
import { useSelectedSpec } from "./context";
import SpecListSection from "./SpecListSection.vue";
import TargetListSection from "./TargetListSection.vue";

const { project } = useCurrentProjectV1();
const { isCreating, plan } = usePlanContext();

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
</script>
