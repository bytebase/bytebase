<template>
  <ReactPageMount
    page="DashboardFrameShell"
    :page-props="{ onReady: handleReady }"
  />

  <teleport v-if="targets.banner" :to="targets.banner">
    <BannersWrapper />
  </teleport>

  <ProvideDashboardContext>
    <teleport v-if="targets.body" :to="targets.body">
      <router-view name="body" />
    </teleport>
  </ProvideDashboardContext>
</template>

<script lang="ts" setup>
import { shallowRef } from "vue";
import BannersWrapper from "@/components/BannersWrapper.vue";
import ProvideDashboardContext from "@/components/ProvideDashboardContext.vue";
import type { DashboardFrameShellTargets } from "@/react/dashboard-shell";
import ReactPageMount from "@/react/ReactPageMount.vue";

const targets = shallowRef<DashboardFrameShellTargets>({
  banner: null,
  body: null,
});

const handleReady = (nextTargets: DashboardFrameShellTargets) => {
  targets.value = nextTargets;
};
</script>
