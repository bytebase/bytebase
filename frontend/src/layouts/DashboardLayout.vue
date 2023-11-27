<template>
  <div class="relative h-screen overflow-hidden flex flex-col">
    <BannersWrapper />
    <!-- Suspense is experimental, be aware of the potential change -->
    <Suspense>
      <template #default>
        <ProvideDashboardContext>
          <router-view name="body" />
        </ProvideDashboardContext>
      </template>
    </Suspense>
    <Suspense>
      <HelpDrawer />
    </Suspense>
  </div>
</template>

<script lang="ts" setup>
import { reactive, watch } from "vue";
import { useRoute } from "vue-router";
import BannersWrapper from "@/components/BannersWrapper.vue";
import HelpDrawer from "@/components/HelpDrawer";
import ProvideDashboardContext from "@/components/ProvideDashboardContext.vue";
import { useHelpStore, useUIStateStore } from "@/store";
import { RouteMapList } from "@/types";

interface LocalState {
  helpTimer: number | undefined;
  RouteMapList: RouteMapList | null;
}

const route = useRoute();
const state = reactive<LocalState>({
  helpTimer: undefined,
  RouteMapList: null,
});

// watch route change for help
watch(
  () => route.name,
  async (routeName) => {
    const uiStateStore = useUIStateStore();
    const helpStore = useHelpStore();

    // Clear timer after every route change.
    if (state.helpTimer) {
      clearTimeout(state.helpTimer);
      state.helpTimer = undefined;
    }

    // Hide opened help drawer if route changed.
    helpStore.exitHelp();

    if (!state.RouteMapList) {
      const res = await fetch("/help/routeMapList.json");
      state.RouteMapList = await res.json();
    }

    const helpId = state.RouteMapList?.find(
      (pair) => pair.routeName === routeName
    )?.helpName;

    if (helpId && !uiStateStore.getIntroStateByKey(`${helpId}`)) {
      state.helpTimer = window.setTimeout(() => {
        helpStore.showHelp(helpId, true);
        uiStateStore.saveIntroStateByKey({
          key: `${helpId}`,
          newState: true,
        });
      }, 1000);
    }
  }
);
</script>
