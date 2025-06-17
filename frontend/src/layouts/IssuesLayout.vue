<template>
  <div class="h-full flex flex-col overflow-hidden">
    <div class="flex-1 flex overflow-hidden">
      <div
        class="flex flex-col min-w-0 flex-1 border-x border-block-border"
        data-label="bb-main-body-wrapper"
      >
        <nav
          class="bg-white border-b border-block-border"
          data-label="bb-dashboard-header"
        >
          <div class="max-w-full mx-auto">
            <DashboardHeader :show-logo="true" />
          </div>
        </nav>

        <!-- This area may scroll -->
        <div
          id="bb-layout-main"
          ref="mainContainerRef"
          class="md:min-w-0 flex-1 overflow-y-auto py-4"
          :class="mainContainerClasses"
        >
          <!-- Start main area-->
          <router-view name="content" />
          <!-- End main area -->
        </div>
      </div>
    </div>

    <Quickstart />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import DashboardHeader from "@/views/DashboardHeader.vue";
import Quickstart from "../components/Quickstart.vue";
import { provideBodyLayoutContext } from "./common";

const mainContainerRef = ref<HTMLDivElement>();

const { mainContainerClasses } = provideBodyLayoutContext({
  mainContainerRef,
});
</script>
