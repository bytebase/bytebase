<template>
  <div class="h-full flex flex-col">
    <div class="border-b">
      <HeaderSection />
    </div>
    <div class="flex-1 flex flex-row">
      <div
        class="flex-1 flex flex-col hide-scrollbar divide-y overflow-x-hidden"
      >
        <SpecListSection />
        <SQLCheckSectionV1 v-if="isCreating" />
        <PlanCheckSection v-else />
        <StatementSection />
        <DescriptionSection />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  HeaderSection,
  PlanCheckSection,
  StatementSection,
  DescriptionSection,
  SQLCheckSectionV1,
  SpecListSection,
} from "./components";
import { providePlanSQLCheckContext } from "./components/SQLCheckSectionV1/context";
import { usePlanContext, usePollPlan } from "./logic";

const { isCreating } = usePlanContext();

usePollPlan();

providePlanSQLCheckContext();
</script>
