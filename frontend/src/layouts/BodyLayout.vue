<template>
  <div class="h-full flex flex-col overflow-hidden">
    <div class="flex-1 flex overflow-hidden">
      <HideInStandaloneMode>
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
            <div
              class="flex-shrink-0 flex border-t border-block-border px-3 py-1.5"
            >
              <div
                v-if="isDemo"
                class="text-sm flex whitespace-nowrap text-accent"
              >
                <heroicons-outline:presentation-chart-bar
                  class="w-5 h-5 mr-1"
                />
                {{ $t("common.demo-mode") }}
              </div>
              <router-link
                v-else-if="!isFreePlan || !hasPermission"
                :to="{ name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION }"
                exact-active-class=""
                class="text-sm flex"
              >
                {{ $t(currentPlan) }}
              </router-link>
              <div
                v-else
                class="text-sm flex whitespace-nowrap mr-1 text-accent cursor-pointer"
                @click="state.showTrialModal = true"
              >
                <heroicons-solid:sparkles class="w-5 h-5" />
                {{ $t(currentPlan) }}
              </div>
              <div
                class="text-sm flex items-center gap-x-1 ml-auto"
                :class="
                  canUpgrade
                    ? 'text-success cursor-pointer'
                    : 'text-control-light cursor-default'
                "
                @click="state.showReleaseModal = canUpgrade"
              >
                <heroicons-outline:volume-up
                  v-if="canUpgrade"
                  class="h-4 w-4"
                />
                {{ version }}
              </div>
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
            <div
              class="flex-shrink-0 flex justify-between border-t border-block-border px-3 py-1.5"
            >
              <div
                v-if="isDemo"
                class="text-sm flex whitespace-nowrap text-accent"
              >
                <heroicons-outline:presentation-chart-bar
                  class="w-5 h-5 mr-1"
                />
                {{ $t("common.demo-mode") }}
              </div>
              <router-link
                v-else-if="!isFreePlan || !hasPermission"
                :to="{ name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION }"
                exact-active-class=""
                class="text-sm flex whitespace-nowrap mr-1"
              >
                {{ $t(currentPlan) }}
              </router-link>
              <div
                v-else-if="subscriptionStore.canTrial"
                class="text-sm flex whitespace-nowrap mr-1 text-accent cursor-pointer"
                @click="state.showTrialModal = true"
              >
                <heroicons-solid:sparkles class="w-5 h-5" />
                {{ $t(currentPlan) }}
              </div>
              <NTooltip>
                <template #trigger>
                  <div
                    class="text-xs flex items-center gap-x-1 whitespace-nowrap"
                    :class="
                      canUpgrade
                        ? 'text-success cursor-pointer'
                        : 'text-control-light cursor-default'
                    "
                    @click="state.showReleaseModal = canUpgrade"
                  >
                    <heroicons-outline:volume-up
                      v-if="canUpgrade"
                      class="h-4 w-4"
                    />
                    {{ version }}
                  </div>
                </template>
                <template #default>
                  <div class="flex flex-col gap-y-1">
                    <div v-if="canUpgrade" class="whitespace-nowrap">
                      {{ $t("settings.release.new-version-available") }}
                    </div>
                    <div>BE Git hash: {{ gitCommitBE }}</div>
                    <div>FE Git hash: {{ gitCommitFE }}</div>
                  </div>
                </template>
              </NTooltip>
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
      </HideInStandaloneMode>

      <div
        class="flex flex-col min-w-0 flex-1"
        :class="pageMode !== 'STANDALONE' && 'border-x border-block-border'"
        data-label="bb-main-body-wrapper"
      >
        <HideInStandaloneMode>
          <nav
            class="bg-white border-b border-block-border"
            data-label="bb-dashboard-header"
          >
            <div class="max-w-full mx-auto">
              <DashboardHeader />
            </div>
          </nav>

          <!-- Static sidebar for mobile -->
          <aside class="md:hidden">
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
        </HideInStandaloneMode>

        <!-- This area may scroll -->
        <div
          id="bb-layout-main"
          ref="mainContainerRef"
          class="md:min-w-0 flex-1 overflow-y-auto py-4"
          :class="mainContainerClasses"
        >
          <HideInStandaloneMode>
            <div class="w-full mx-auto md:flex">
              <div class="md:min-w-0 md:flex-1">
                <div
                  class="w-full flex flex-row justify-between items-center flex-wrap px-4 gap-x-4"
                >
                  <QuickActionPanel
                    :quick-action-list="quickActionList"
                    class="flex-1 pb-4"
                  />
                </div>
              </div>
            </div>
          </HideInStandaloneMode>
          <!-- Start main area-->
          <router-view name="content" />
          <!-- End main area -->
        </div>
      </div>
    </div>

    <HideInStandaloneMode>
      <Quickstart />
    </HideInStandaloneMode>
  </div>

  <TrialModal
    v-if="state.showTrialModal"
    @cancel="state.showTrialModal = false"
  />
  <HideInStandaloneMode>
    <ReleaseRemindModal
      v-if="state.showReleaseModal"
      @cancel="state.showReleaseModal = false"
    />
  </HideInStandaloneMode>
</template>

<script lang="ts" setup>
import { useMounted, useWindowSize } from "@vueuse/core";
import { storeToRefs } from "pinia";
import { computed, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import HideInStandaloneMode from "@/components/misc/HideInStandaloneMode.vue";
import { SETTING_ROUTE_WORKSPACE_SUBSCRIPTION } from "@/router/dashboard/workspaceSetting";
import {
  useActuatorV1Store,
  useCurrentUserV1,
  useSubscriptionV1Store,
} from "@/store";
import type { QuickActionType } from "@/types";
import { QuickActionPermissionMap } from "@/types";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import DashboardHeader from "@/views/DashboardHeader.vue";
import QuickActionPanel from "../components/QuickActionPanel.vue";
import Quickstart from "../components/Quickstart.vue";
import { provideBodyLayoutContext } from "./common";

interface LocalState {
  showMobileOverlay: boolean;
  showTrialModal: boolean;
  showReleaseModal: boolean;
}

const actuatorStore = useActuatorV1Store();
const subscriptionStore = useSubscriptionV1Store();
const router = useRouter();
const mounted = useMounted();

const state = reactive<LocalState>({
  showMobileOverlay: false,
  showTrialModal: false,
  showReleaseModal: false,
});

const currentUserV1 = useCurrentUserV1();
const mainContainerRef = ref<HTMLDivElement>();
const { width: windowWidth } = useWindowSize();

const sidebarView = computed(() => {
  return windowWidth.value >= 768 ? "DESKTOP" : "MOBILE";
});

const hasPermission = computed(() =>
  hasWorkspacePermissionV2(currentUserV1.value, "bb.settings.set")
);

const { pageMode, isDemo } = storeToRefs(actuatorStore);

actuatorStore.tryToRemindRelease().then((openRemindModal) => {
  state.showReleaseModal = openRemindModal;
});

const canUpgrade = computed(() => {
  return actuatorStore.hasNewRelease;
});

const getQuickActionList = (list: QuickActionType[]): QuickActionType[] => {
  return list.filter((action) => {
    if (!QuickActionPermissionMap.has(action)) {
      return false;
    }
    return QuickActionPermissionMap.get(action)?.every((permission) =>
      hasWorkspacePermissionV2(currentUserV1.value, permission)
    );
  });
};

const quickActionList = computed(() => {
  const quickActionListFunc = router.currentRoute.value.meta.getQuickActionList;
  const listByRole = quickActionListFunc
    ? quickActionListFunc(router.currentRoute.value)
    : [];

  return getQuickActionList(listByRole);
});

const version = computed(() => {
  const v = actuatorStore.version;
  if (v.split(".").length == 3) {
    return `v${v}`;
  }
  return v;
});

const gitCommitBE = computed(() => {
  return `${actuatorStore.gitCommit.substring(0, 7)}`;
});
const gitCommitFE = computed(() => {
  const commitHash = import.meta.env.BB_GIT_COMMIT_ID_FE as string | undefined;
  return (commitHash ?? "unknown").substring(0, 7);
});

const currentPlan = computed((): string => {
  const plan = subscriptionStore.currentPlan;
  switch (plan) {
    case PlanType.TEAM:
      return "subscription.plan.team.title";
    case PlanType.ENTERPRISE:
      return "subscription.plan.enterprise.title";
    default:
      if (hasPermission.value) {
        return "subscription.plan.try";
      }
      return "subscription.plan.free.title";
  }
});

const isFreePlan = computed((): boolean => {
  const plan = subscriptionStore.currentPlan;
  return plan === PlanType.FREE;
});

const { mainContainerClasses } = provideBodyLayoutContext({
  mainContainerRef,
});
</script>
