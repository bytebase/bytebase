<template>
  <div class="h-full flex flex-col overflow-hidden">
    <div class="flex-1 flex overflow-hidden">
      <template v-if="!isRootPath">
        <!-- Off-canvas menu for mobile, show/hide based on off-canvas menu state. -->
        <div
          v-show="state.showMobileOverlay"
          :class="[sidebarView === 'MOBILE' ? 'block' : 'hidden']"
          class="fixed inset-0 flex z-40"
        >
          <div class="fixed inset-0">
            <div
              class="absolute inset-0 bg-gray-600 opacity-75"
              @click.prevent="state.showMobileOverlay = false"
            ></div>
          </div>
          <div
            tabindex="0"
            class="relative flex-1 flex flex-col max-w-xs w-full bg-white focus:outline-hidden"
          >
            <div class="absolute top-0 right-0 -mr-12 pt-2">
              <button
                type="button"
                class="ml-1 flex items-center justify-center h-10 w-10 rounded-full focus:outline-hidden focus:ring-2 focus:ring-inset focus:ring-white"
                @click.prevent="state.showMobileOverlay = false"
              >
                <span class="sr-only">Close sidebar</span>
                <!-- Heroicon name: x -->
                <heroicons-solid:x class="h-6 w-6 text-white" />
              </button>
            </div>
            <!-- Mobile Sidebar -->
            <div id="sidebar-mobile" class="flex-1 h-0 py-0 overflow-y-auto">
              <!-- Empty as teleport placeholder -->
            </div>
          </div>
          <div class="shrink-0 w-14" aria-hidden="true">
            <!-- Force sidebar to shrink to fit close icon -->
          </div>
        </div>

        <!-- Static sidebar for desktop -->
        <aside
          class="shrink-0"
          data-label="bb-dashboard-static-sidebar"
          :class="[sidebarView === 'DESKTOP' ? 'flex' : 'hidden']"
        >
          <div class="flex flex-col w-52 bg-control-bg">
            <!-- Sidebar component, swap this element with another sidebar if you like -->
            <div
              id="sidebar-desktop"
              class="flex-1 flex flex-col py-0 overflow-y-auto"
            >
              <!-- Empty as teleport placeholder -->
            </div>
          </div>
        </aside>

        <!--
          Do not render two instances of sidebars to desktop and mobile sidebars
          but render one sidebar and teleport it to a correct container when
          screen size varies
        -->
        <teleport
          v-if="mounted"
          :to="
            sidebarView === 'DESKTOP' ? '#sidebar-desktop' : '#sidebar-mobile'
          "
        >
          <router-view name="leftSidebar" />
        </teleport>
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
            <DashboardHeader :show-logo="false" />
          </div>
        </nav>

        <aside v-if="!isRootPath" class="md:hidden">
          <!-- Static sidebar for mobile -->
          <div
            class="flex items-center justify-start bg-gray-50 border-b border-block-border px-4"
          >
            <div>
              <button
                type="button"
                class="-mr-3 h-8 w-8 inline-flex items-center justify-center rounded-md text-gray-500 hover:text-gray-900"
                @click.prevent="state.showMobileOverlay = true"
              >
                <span class="sr-only">Open sidebar</span>
                <!-- Heroicon name: menu -->
                <heroicons-outline:menu class="h-4 w-4" />
              </button>
            </div>
          </div>
        </aside>

        <!-- This area may scroll -->
        <div
          id="bb-layout-main"
          ref="mainContainerRef"
          class="md:min-w-0 flex-1 overflow-y-auto py-4"
          :class="mainContainerClasses"
        >
          <RoutePermissionGuard
            class="mx-4"
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
import { useMounted, useWindowSize } from "@vueuse/core";
import { computed, onMounted, onUnmounted, reactive, ref } from "vue";
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
const mounted = useMounted();

const state = reactive<LocalState>({
  showMobileOverlay: false,
  showReleaseModal: false,
  showRefreshRemindModal: false,
});

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

const { mainContainerClasses } = provideBodyLayoutContext({
  mainContainerRef,
});
</script>
