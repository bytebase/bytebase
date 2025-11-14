<template>
  <div
    ref="containerRef"
    class="h-full flex flex-col items-stretch justify-between overflow-hidden text-sm p-1"
  >
    <div class="flex flex-col gap-y-1">
      <div class="flex flex-col justify-center items-center pb-1">
        <router-link target="_blank" rel="noopener noreferrer" :to="linkTarget">
          <img
            class="w-[36px] h-auto"
            src="@/assets/logo-icon.svg"
            alt="Bytebase"
          />
        </router-link>
      </div>
      <div class="w-full h-0 border-t" />
      <TabItem
        tab="WORKSHEET"
        :size="size"
        @click="handleClickTab('WORKSHEET')"
      />
      <TabItem tab="SCHEMA" :size="size" @click="handleClickTab('SCHEMA')" />
      <TabItem tab="HISTORY" :size="size" @click="handleClickTab('HISTORY')" />
    </div>

    <div class="flex flex-col justify-end items-center"></div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import { PROJECT_V1_ROUTE_DETAIL } from "@/router/dashboard/projectV1";
import { WORKSPACE_ROUTE_LANDING } from "@/router/dashboard/workspaceRoutes";
import { type AsidePanelTab, useSQLEditorContext } from "../../context";
import { type Size } from "./common";
import TabItem from "./TabItem.vue";

withDefaults(
  defineProps<{
    size?: Size;
  }>(),
  {
    size: "medium",
  }
);

const route = useRoute();
const { asidePanelTab } = useSQLEditorContext();

const linkTarget = computed(() => {
  // If we have a project in the route, navigate to that project's detail page
  const project = route.params.project as string | undefined;
  if (project) {
    return {
      name: PROJECT_V1_ROUTE_DETAIL,
      params: {
        projectId: project,
      },
    };
  }
  // Otherwise fallback to workspace landing
  return {
    name: WORKSPACE_ROUTE_LANDING,
  };
});

const handleClickTab = (target: AsidePanelTab) => {
  asidePanelTab.value = target;
};
</script>
