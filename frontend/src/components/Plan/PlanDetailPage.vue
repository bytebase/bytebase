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

        <SQLCheckSection v-if="isCreating" @update:advices="advices = $event" />
        <PlanCheckSection v-if="!isCreating" />

        <StatementSection :advices="advices" />
        <DescriptionSection />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import type { Advice } from "@/types/proto/v1/sql_service";
import { provideSQLCheckContext } from "../SQLCheck";
import {
  HeaderSection,
  PlanCheckSection,
  StatementSection,
  DescriptionSection,
  SQLCheckSection,
  SpecListSection,
} from "./components";
import { usePlanContext, usePollPlan } from "./logic";

const { isCreating } = usePlanContext();
const advices = ref<Advice[]>();

usePollPlan();

provideSQLCheckContext();
</script>
