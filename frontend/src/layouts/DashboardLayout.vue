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
    <Suspense v-if="!hideHelp">
      <HelpDrawer />
    </Suspense>
  </div>
</template>

<script lang="ts" setup>
import { reactive, watch, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import BannersWrapper from "@/components/BannersWrapper.vue";
import HelpDrawer from "@/components/HelpDrawer";
import ProvideDashboardContext from "@/components/ProvideDashboardContext.vue";
import { WORKSPACE_HOME_MODULE } from "@/router/dashboard/workspaceRoutes";
import { useRecentVisit } from "@/router/useRecentVisit";
import { useAppFeature, useHelpStore, useUIStateStore } from "@/store";
import type { RouteMapList } from "@/types";
import { isDev } from "@/utils";

interface LocalState {
  helpTimer: number | undefined;
  RouteMapList: RouteMapList | null;
}

const route = useRoute();
const router = useRouter();
const state = reactive<LocalState>({
  helpTimer: undefined,
  RouteMapList: null,
});
const { lastVisit } = useRecentVisit();
const hideHelp = useAppFeature("bb.feature.hide-help");

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

onMounted(() => {
  if (
    isDev() &&
    lastVisit.value?.path &&
    route.name?.toString() === WORKSPACE_HOME_MODULE
  ) {
    router.replace({
      path: lastVisit.value?.path,
    });
  }
});
</script>
