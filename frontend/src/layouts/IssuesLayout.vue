<template>
  <ReactPageMount
    page="DashboardBodyShell"
    :page-props="{
      variant: 'issues',
      routeKey: route.fullPath,
      onReady: handleReady,
    }"
  />

  <teleport v-if="contentTarget" :to="contentTarget">
    <router-view name="content" />
  </teleport>

  <teleport v-if="quickstartTarget" :to="quickstartTarget">
    <Quickstart />
  </teleport>
</template>

<script lang="ts" setup>
import { computed, ref, shallowRef } from "vue";
import { useRoute } from "vue-router";
import Quickstart from "@/components/Quickstart.vue";
import type { DashboardShellTargets } from "@/react/dashboard-shell";
import ReactPageMount from "@/react/ReactPageMount.vue";
import { provideBodyLayoutContext } from "./common";

const route = useRoute();
const shellTargets = shallowRef<DashboardShellTargets>({
  desktopSidebar: null,
  mobileSidebar: null,
  content: null,
  quickstart: null,
  mainContainer: null,
});
const mainContainerRef = ref<HTMLDivElement>();

const contentTarget = computed(() => shellTargets.value.content);
const quickstartTarget = computed(() => shellTargets.value.quickstart);

const handleReady = (targets: DashboardShellTargets) => {
  shellTargets.value = targets;
  mainContainerRef.value = targets.mainContainer ?? undefined;
};

provideBodyLayoutContext({
  mainContainerRef,
});
</script>
