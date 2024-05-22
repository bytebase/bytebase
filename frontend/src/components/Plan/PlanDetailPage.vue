<template>
  <div class="h-full flex flex-col">
    <div class="border-b">
      <HeaderSection />
    </div>
    <div class="flex-1 flex flex-row">
      <div
        class="flex-1 flex flex-col hide-scrollbar divide-y overflow-x-hidden"
      >
        <StepSection />
        <SpecListSection />

        <SQLCheckSection v-if="isCreating" />
        <PlanCheckSection v-if="!isCreating" />

        <StatementSection />
        <DescriptionSection />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { provideSQLCheckContext } from "../SQLCheck";
import {
  HeaderSection,
  PlanCheckSection,
  StatementSection,
  DescriptionSection,
  SQLCheckSection,
  StepSection,
  SpecListSection,
} from "./components";
import { usePlanContext, usePollPlan } from "./logic";

const { isCreating } = usePlanContext();

usePollPlan();

provideSQLCheckContext();
</script>
