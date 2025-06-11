<template>
  <div
    class="flex-1 flex flex-col hide-scrollbar gap-3 divide-y overflow-x-hidden"
  >
    <TargetListSection />
    <SQLCheckV1Section v-if="isCreating" />
    <PlanCheckSection v-else />
    <div class="w-full space-y-3 pt-3">
      <StatementSection />
      <Configuration />
    </div>
  </div>
</template>

<script setup lang="ts">
import { usePlanContext } from "../../logic/context";
import Configuration from "../Configuration";
import PlanCheckSection from "../PlanCheckSection";
import SQLCheckV1Section from "../SQLCheckV1Section";
import StatementSection from "../StatementSection";
import TargetListSection from "./TargetListSection.vue";

const { isCreating } = usePlanContext();
</script>
