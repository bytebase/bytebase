<template>
  <div ref="containerRef" class="h-full flex flex-col">
    <div class="border-b">
      <HeaderSection />
    </div>
    <div class="w-full flex-1 flex flex-row">
      <Navigator />
      <div
        class="flex-1 flex flex-col hide-scrollbar divide-y overflow-x-hidden"
      >
        <Overview v-if="!selectedSpec" />
        <SpecDetailView v-else :key="selectedSpec.id" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import {
  HeaderSection,
  Navigator,
  Overview,
  SpecDetailView,
} from "./components";
import { usePlanContext, usePollPlan } from "./logic";

const { selectedSpec } = usePlanContext();
const containerRef = ref<HTMLElement>();

usePollPlan();
</script>
