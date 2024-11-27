<template>
  <div class="h-full flex flex-col overflow-hidden">
    <div class="flex-1 flex overflow-hidden">
      <template v-if="!hideSidebar && !isRootPath">
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
            class="relative flex-1 flex flex-col max-w-xs w-full bg-white focus:outline-none"
          >
            <div class="absolute top-0 right-0 -mr-12 pt-2">
              <button
                type="button"
                class="ml-1 flex items-center justify-center h-10 w-10 rounded-full focus:outline-none focus:ring-2 focus:ring-inset focus:ring-white"
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
          <div class="flex-shrink-0 w-14" aria-hidden="true">
            <!-- Force sidebar to shrink to fit close icon -->
          </div>
        </div>

        <!-- Static sidebar for desktop -->
        <aside
          class="flex-shrink-0"
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
        :class="!hideHeader && 'border-x border-block-border'"
        data-label="bb-main-body-wrapper"
      >
        <nav
          v-if="!hideHeader && !isRootPath"
          class="bg-white border-b border-block-border"
          data-label="bb-dashboard-header"
        >
          <div class="max-w-full mx-auto">
            <DashboardHeader :show-logo="false" />
          </div>
        </nav>

        <aside v-if="!hideSidebar && !isRootPath" class="md:hidden">
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
          <!-- Start main area-->
          <router-view name="content" />
          <!-- End main area -->
        </div>
      </div>
    </div>

    <Quickstart v-if="!hideQuickStart" />
  </div>

  <TrialModal
    v-if="state.showTrialModal"
    @cancel="state.showTrialModal = false"
  />
  <ReleaseRemindModal
    v-if="
      !hideReleaseRemind &&
      state.showReleaseModal &&
      route.name !== WORKSPACE_ROOT_MODULE
    "
    @cancel="state.showReleaseModal = false"
  />
</template>

<script lang="ts" setup>
import { useMounted, useWindowSize } from "@vueuse/core";
import { computed, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import ReleaseRemindModal from "@/components/ReleaseRemindModal.vue";
import TrialModal from "@/components/TrialModal.vue";
import { WORKSPACE_ROOT_MODULE } from "@/router/dashboard/workspaceRoutes";
import { useActuatorV1Store, useAppFeature } from "@/store";
import DashboardHeader from "@/views/DashboardHeader.vue";
import Quickstart from "../components/Quickstart.vue";
import { provideBodyLayoutContext } from "./common";

interface LocalState {
  showMobileOverlay: boolean;
  showTrialModal: boolean;
  showReleaseModal: boolean;
}

const actuatorStore = useActuatorV1Store();
const route = useRoute();
const router = useRouter();
const mounted = useMounted();

const state = reactive<LocalState>({
  showMobileOverlay: false,
  showTrialModal: false,
  showReleaseModal: false,
});

const mainContainerRef = ref<HTMLDivElement>();
const { width: windowWidth } = useWindowSize();

const isRootPath = computed(() => {
  return router.currentRoute.value.name === WORKSPACE_ROOT_MODULE;
});

const sidebarView = computed(() => {
  return windowWidth.value >= 768 ? "DESKTOP" : "MOBILE";
});

const hideSidebar = useAppFeature("bb.feature.console.hide-sidebar");
const hideHeader = useAppFeature("bb.feature.console.hide-header");
const hideQuickStart = useAppFeature("bb.feature.hide-quick-start");
const hideReleaseRemind = useAppFeature("bb.feature.hide-release-remind");

actuatorStore.tryToRemindRelease().then((openRemindModal) => {
  state.showReleaseModal = openRemindModal;
});

const { mainContainerClasses } = provideBodyLayoutContext({
  mainContainerRef,
});
</script>
