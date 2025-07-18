<template>
  <div
    class="flex-1 flex flex-col hide-scrollbar gap-3 divide-y overflow-x-hidden"
  >
    <div class="w-full flex flex-col px-4 gap-2">
      <SpecListSection v-if="isCreating || plan.specs.length > 1" />
      <TargetListSection />
    </div>
    <SQLCheckV1Section v-if="isCreating" />
    <PlanCheckSection v-else />
    <div class="w-full space-y-3 pt-3">
      <StatementSection />
      <Configuration />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Ref } from "vue";
import { useCurrentProjectV1 } from "@/store";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { usePlanContext } from "../../logic/context";
import Configuration from "../Configuration";
import PlanCheckSection from "../PlanCheckSection";
import { providePlanSQLCheckContext } from "../SQLCheckSection";
import SQLCheckV1Section from "../SQLCheckV1Section";
import StatementSection from "../StatementSection";
import SpecListSection from "./SpecListSection.vue";
import TargetListSection from "./TargetListSection.vue";
import { useSelectedSpec } from "./context";

const { project } = useCurrentProjectV1();
const { isCreating, plan } = usePlanContext();

const selectedSpec = useSelectedSpec();

providePlanSQLCheckContext({
  project,
  plan,
  selectedSpec: selectedSpec as Ref<Plan_Spec>,
});
</script>
