<template>
  <div class="h-full flex flex-col overflow-hidden">
    <div class="flex-1 flex overflow-hidden">
      <template v-if="!isRootPath">
        <!-- Mobile sidebar drawer -->
        <NDrawer
          v-model:show="state.showMobileOverlay"
          placement="left"
          :width="208"
          :class="[sidebarView === 'MOBILE' ? '' : 'hidden!']"
        >
          <NDrawerContent body-content-class="!p-0">
            <div class="h-full overflow-y-auto bg-control-bg">
              <router-view v-if="sidebarView === 'MOBILE'" name="leftSidebar" />
            </div>
          </NDrawerContent>
        </NDrawer>

        <!-- Static sidebar for desktop -->
        <aside
          v-if="sidebarView === 'DESKTOP'"
          class="shrink-0 flex"
          data-label="bb-dashboard-static-sidebar"
        >
          <div class="flex flex-col w-52 bg-control-bg">
            <div class="flex-1 flex flex-col py-0 overflow-y-auto">
              <router-view name="leftSidebar" />
            </div>
          </div>
        </aside>
      </template>

      <div
        class="flex flex-col min-w-0 flex-1"
        data-label="bb-main-body-wrapper"
      >
        <nav
          v-if="!isRootPath"
          class="bg-white border-b border-block-border"
          data-label="bb-dashboard-header"
        >
          <div class="max-w-full mx-auto">
            <DashboardHeader
              :show-logo="false"
              :show-mobile-sidebar-toggle="true"
              @toggle-mobile-sidebar="state.showMobileOverlay = true"
            />
          </div>
        </nav>

        <!-- This area may scroll -->
        <div
          id="bb-layout-main"
          ref="mainContainerRef"
          class="md:min-w-0 flex-1 overflow-y-auto"
        >
          <RoutePermissionGuard
            :routes="[
              ...workspaceRoutes,
              ...workspaceSettingRoutes,
              ...environmentV1Routes,
              ...instanceRoutes,
            ]"
          >
            <!-- Start main area-->
            <router-view name="content" />
            <!-- End main area -->
          </RoutePermissionGuard>
        </div>
      </div>
    </div>

    <Quickstart />
  </div>

  <ReleaseRemindModal
    v-if="state.showReleaseModal && route.name !== WORKSPACE_ROOT_MODULE"
    @cancel="state.showReleaseModal = false"
  />
</template>

<script lang="ts" setup>
import { useWindowSize } from "@vueuse/core";
import { NDrawer, NDrawerContent } from "naive-ui";
import { computed, onMounted, onUnmounted, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import RoutePermissionGuard from "@/components/Permission/RoutePermissionGuard.vue";
import ReleaseRemindModal from "@/components/ReleaseRemindModal.vue";
import { t } from "@/plugins/i18n";
import environmentV1Routes from "@/router/dashboard/environmentV1";
import instanceRoutes from "@/router/dashboard/instance";
import workspaceRoutes from "@/router/dashboard/workspace";
import { WORKSPACE_ROOT_MODULE } from "@/router/dashboard/workspaceRoutes";
import workspaceSettingRoutes from "@/router/dashboard/workspaceSetting";
import {
  pushNotification,
  useActuatorV1Store,
  usePermissionStore,
  useSubscriptionV1Store,
} from "@/store";
import { PresetRoleType } from "@/types";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import DashboardHeader from "@/views/DashboardHeader.vue";
import Quickstart from "../components/Quickstart.vue";
import { provideBodyLayoutContext } from "./common";

interface LocalState {
  showMobileOverlay: boolean;
  showReleaseModal: boolean;
  showRefreshRemindModal: boolean;
}

const actuatorStore = useActuatorV1Store();
const permissionStore = usePermissionStore();
const subscriptionStore = useSubscriptionV1Store();
const route = useRoute();
const router = useRouter();

const state = reactive<LocalState>({
  showMobileOverlay: false,
  showReleaseModal: false,
  showRefreshRemindModal: false,
});

// Close mobile drawer on route change
watch(
  () => route.fullPath,
  () => {
    state.showMobileOverlay = false;
  }
);

const mainContainerRef = ref<HTMLDivElement>();
const { width: windowWidth } = useWindowSize();

const isRootPath = computed(() => {
  return router.currentRoute.value.name === WORKSPACE_ROOT_MODULE;
});

const sidebarView = computed(() => {
  return windowWidth.value >= 768 ? "DESKTOP" : "MOBILE";
});

actuatorStore.tryToRemindRelease().then((openRemindModal) => {
  if (
    subscriptionStore.currentPlan === PlanType.ENTERPRISE &&
    !permissionStore.currentRolesInWorkspace.has(PresetRoleType.WORKSPACE_ADMIN)
  ) {
    return;
  }
  state.showReleaseModal = openRemindModal;
});

// compare BE and FE commit hash every 30 minutes.
// if they are different, show the refresh remind modal.
const refreshRemindTimer = ref<NodeJS.Timeout>();
onMounted(async () => {
  const remind = await actuatorStore.tryToRemindRefresh();
  if (remind) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: t("refresh-remind.title"),
      description: t("refresh-remind.description"),
      manualHide: true,
    });
  }
  refreshRemindTimer.value = setInterval(
    async () => {
      const remind = await actuatorStore.tryToRemindRefresh();
      if (remind) {
        pushNotification({
          module: "bytebase",
          style: "WARN",
          title: t("refresh-remind.title"),
          description: t("refresh-remind.description"),
          manualHide: true,
        });
      }
    },
    1000 * 60 * 30
  );
});
onUnmounted(() => {
  if (refreshRemindTimer.value) {
    clearInterval(refreshRemindTimer.value);
  }
});

provideBodyLayoutContext({
  mainContainerRef,
});
</script>
