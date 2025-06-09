<template>
  <div class="flex-1 flex flex-col hide-scrollbar divide-y overflow-x-hidden">
    <SQLCheckSection v-if="isCreating" />
    <SpecBasedPlanCheckSection v-else />
    <TargetListSection />
    <div class="w-full">
      <StatementSection />
      <Configuration />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Ref } from "vue";
import { useCurrentProjectV1 } from "@/store";
import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import { usePlanContext } from "../../logic/context";
import Configuration from "../Configuration";
import SpecBasedPlanCheckSection from "../PlanCheckSection/SpecBasedPlanCheckSection.vue";
import SQLCheckSection, {
  providePlanSQLCheckContext,
} from "../SQLCheckSection";
import StatementSection from "../StatementSection";
import TargetListSection from "./TargetListSection.vue";

const { project } = useCurrentProjectV1();
const { isCreating, plan, selectedSpec } = usePlanContext();

providePlanSQLCheckContext({
  project,
  plan,
  selectedSpec: selectedSpec as Ref<Plan_Spec>,
});
</script>
