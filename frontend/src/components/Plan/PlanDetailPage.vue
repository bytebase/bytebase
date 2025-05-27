<template>
  <div ref="containerRef" class="h-full flex flex-col">
    <div class="border-b">
      <HeaderSection />
    </div>
    <div class="flex-1 flex flex-row">
      <div
        class="flex-1 flex flex-col hide-scrollbar divide-y overflow-x-hidden"
      >
        <SpecListSection />
        <SQLCheckSection v-if="isCreating" />
        <PlanCheckSection v-else />
        <StatementSection />
        <DescriptionSection />
      </div>
      <div
        v-if="sidebarMode == 'DESKTOP'"
        class="hide-scrollbar border-l"
        :style="{
          width: `${desktopSidebarWidth}px`,
        }"
      >
        <Sidebar />
      </div>
    </div>
  </div>

  <template v-if="sidebarMode === 'MOBILE'">
    <!-- mobile sidebar -->
    <Drawer :show="mobileSidebarOpen" @close="mobileSidebarOpen = false">
      <div
        style="
          min-width: 240px;
          width: 80vw;
          max-width: 320px;
          padding: 0.5rem 0;
        "
      >
        <Sidebar v-if="sidebarMode === 'MOBILE'" />
      </div>
    </Drawer>
  </template>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { Drawer } from "@/components/v2";
import { useCurrentProjectV1 } from "@/store";
import {
  HeaderSection,
  PlanCheckSection,
  StatementSection,
  DescriptionSection,
  SQLCheckSection,
  SpecListSection,
} from "./components";
import { providePlanSQLCheckContext } from "./components/SQLCheckSection";
import Sidebar from "./components/Sidebar";
import { usePlanContext, usePollPlan } from "./logic";
import { provideSidebarContext } from "./logic";

const { project } = useCurrentProjectV1();
const { isCreating, plan, selectedSpec } = usePlanContext();
const containerRef = ref<HTMLElement>();

usePollPlan();

providePlanSQLCheckContext({
  project,
  plan: plan,
  selectedSpec: selectedSpec,
});

const {
  mode: sidebarMode,
  desktopSidebarWidth,
  mobileSidebarOpen,
} = provideSidebarContext(containerRef);
</script>
