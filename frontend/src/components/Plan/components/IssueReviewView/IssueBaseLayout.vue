<template>
  <div class="flex-1 flex w-full border-t -mt-px">
    <!-- Left Panel - Activity -->
    <div class="flex-1 shrink p-4 flex flex-col gap-y-4 overflow-x-auto">
      <slot />
      <ActivitySection />
    </div>

    <!-- Desktop Sidebar -->
    <div
      v-if="sidebarMode === 'DESKTOP'"
      class="shrink-0 flex flex-col border-l"
      :style="{
        width: `${desktopSidebarWidth}px`,
      }"
    >
      <Sidebar />
    </div>

    <!-- Mobile Sidebar -->
    <template v-if="sidebarMode === 'MOBILE'">
      <Drawer :show="mobileSidebarOpen" @close="mobileSidebarOpen = false">
        <div
          style="
            min-width: 240px;
            width: 80vw;
            max-width: 320px;
            padding: 0.5rem;
          "
        >
          <Sidebar />
        </div>
      </Drawer>
    </template>
  </div>
</template>

<script setup lang="ts">
import { Drawer } from "@/components/v2";
import { usePlanContextWithIssue } from "../..";
import { useSidebarContext } from "../../logic/sidebar";
import { ActivitySection } from "./ActivitySection";
import { Sidebar } from "./Sidebar";

usePlanContextWithIssue();

const {
  mode: sidebarMode,
  desktopSidebarWidth,
  mobileSidebarOpen,
} = useSidebarContext();
</script>
