<template>
  <div
    ref="containerRef"
    class="h-full flex flex-col items-stretch justify-between overflow-hidden text-sm p-1"
  >
    <div class="flex flex-col gap-y-1">
      <div class="flex flex-col justify-center items-center pb-1">
        <router-link
          target="_blank"
          rel="noopener noreferrer"
          :to="{
            name: WORKSPACE_ROUTE_LANDING,
          }"
        >
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
import { WORKSPACE_ROUTE_LANDING } from "@/router/dashboard/workspaceRoutes";
import { useSQLEditorContext, type AsidePanelTab } from "../../context";
import TabItem from "./TabItem.vue";
import { type Size } from "./common";

withDefaults(
  defineProps<{
    size?: Size;
  }>(),
  {
    size: "medium",
  }
);

const { asidePanelTab } = useSQLEditorContext();

const handleClickTab = (target: AsidePanelTab) => {
  asidePanelTab.value = target;
};
</script>
