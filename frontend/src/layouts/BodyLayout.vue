<template>
  <div class="h-full flex flex-col overflow-hidden">
    <div class="flex-1 flex overflow-hidden">
      <!-- Off-canvas menu for mobile, show/hide based on off-canvas menu state. -->
      <div v-if="state.showMobileOverlay" class="md:hidden">
        <div class="fixed inset-0 flex z-40">
          <div class="fixed inset-0">
            <div class="absolute inset-0 bg-gray-600 opacity-75"></div>
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
            <div class="flex-1 h-0 py-4 overflow-y-auto">
              <router-view name="leftSidebar" />
            </div>
            <router-link
              to="/archive"
              class="outline-item group flex items-center px-4 py-2"
            >
              <heroicons-outline:archive class="w-5 h-5 mr-2" />
              {{ $t("common.archive") }}
            </router-link>
            <div
              class="flex-shrink-0 flex border-t border-block-border px-3 py-2"
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
                to="/setting/subscription"
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
                class="text-sm flex items-center gap-x-1 ml-auto tooltip-wrapper"
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
                <span v-if="gitCommit" class="tooltip"
                  >Git hash {{ gitCommit }}</span
                >
              </div>
            </div>
          </div>
          <div class="flex-shrink-0 w-14" aria-hidden="true">
            <!-- Force sidebar to shrink to fit close icon -->
          </div>
        </div>
      </div>

      <!-- Static sidebar for desktop -->
      <aside
        class="hidden md:flex md:flex-shrink-0"
        data-label="bb-dashboard-static-sidebar"
      >
        <div class="flex flex-col w-52 bg-control-bg">
          <!-- Sidebar component, swap this element with another sidebar if you like -->
          <div class="flex-1 flex flex-col py-0 overflow-y-auto">
            <router-view name="leftSidebar" />
          </div>
          <router-link
            to="/archive"
            class="outline-item group flex items-center px-4 py-2"
          >
            <heroicons-outline:archive class="w-5 h-5 mr-2" />
            {{ $t("common.archive") }}
          </router-link>
          <div
            class="flex-shrink-0 flex justify-between border-t border-block-border px-3 py-2"
          >
            <div
              v-if="isDemo"
              class="text-sm flex whitespace-nowrap text-accent"
            >
              <heroicons-outline:presentation-chart-bar class="w-5 h-5 mr-1" />
              {{ $t("common.demo-mode") }}
            </div>
            <router-link
              v-else-if="!isFreePlan || !hasPermission"
              to="/setting/subscription"
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
            <div
              class="text-xs flex items-center gap-x-1 tooltip-wrapper whitespace-nowrap"
              :class="
                canUpgrade
                  ? 'text-success cursor-pointer'
                  : 'text-control-light cursor-default'
              "
              @click="state.showReleaseModal = canUpgrade"
            >
              <heroicons-outline:volume-up v-if="canUpgrade" class="h-4 w-4" />
              {{ version }}
              <span v-if="canUpgrade" class="tooltip whitespace-nowrap">
                {{ $t("settings.release.new-version-available") }}
              </span>
              <span v-else-if="gitCommit" class="tooltip">
                Git hash {{ gitCommit }}
              </span>
            </div>
          </div>
        </div>
      </aside>
      <div
        class="flex flex-col min-w-0 flex-1 border-l border-r border-block-border"
        data-label="bb-main-body-wrapper"
      >
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
            class="flex items-center justify-start bg-gray-50 border-b border-block-border px-4 py-1.5"
          >
            <div>
              <button
                type="button"
                class="-mr-3 h-12 w-12 inline-flex items-center justify-center rounded-md text-gray-500 hover:text-gray-900"
                @click.prevent="state.showMobileOverlay = true"
              >
                <span class="sr-only">Open sidebar</span>
                <!-- Heroicon name: menu -->
                <heroicons-outline:menu class="h-6 w-6" />
              </button>
            </div>
            <div v-if="showBreadcrumb" class="ml-4">
              <Breadcrumb />
            </div>
          </div>
        </aside>

        <div class="w-full mx-auto md:flex">
          <div class="md:min-w-0 md:flex-1">
            <div v-if="showBreadcrumb" class="hidden md:block px-4 pt-4">
              <Breadcrumb />
            </div>
            <div
              class="w-full flex flex-row justify-between items-center flex-wrap px-4 gap-x-4"
            >
              <div v-if="quickActionList.length > 0" class="flex-1 pt-6 pb-2">
                <QuickActionPanel :quick-action-list="quickActionList" />
              </div>
              <div
                v-if="route.name === 'workspace.home'"
                class="hidden md:flex"
              >
                <a
                  href="/sql-editor"
                  target="_blank"
                  class="btn-normal items-center !px-4 !text-base"
                >
                  <heroicons-solid:terminal class="text-accent w-6 h-6 mr-2" />
                  <span class="whitespace-nowrap">{{
                    $t("sql-editor.self")
                  }}</span>
                </a>
              </div>
            </div>
          </div>
        </div>
        <!-- This area may scroll -->
        <div
          class="md:min-w-0 md:flex-1 overflow-y-auto"
          :class="showBreadcrumb || quickActionList.length > 0 ? 'mt-2' : ''"
        >
          <!-- Start main area-->
          <router-view name="content" />
          <!-- End main area -->
        </div>
      </div>
    </div>

    <Quickstart />
  </div>

  <TrialModal
    v-if="state.showTrialModal"
    @cancel="state.showTrialModal = false"
  />
  <ReleaseRemindModal
    v-if="state.showReleaseModal"
    @cancel="state.showReleaseModal = false"
  />
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { computed, reactive } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  useActuatorV1Store,
  useCurrentUserV1,
  useSubscriptionV1Store,
} from "@/store";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { hasWorkspacePermissionV1 } from "@/utils";
import DashboardHeader from "@/views/DashboardHeader.vue";
import Breadcrumb from "../components/Breadcrumb.vue";
import QuickActionPanel from "../components/QuickActionPanel.vue";
import Quickstart from "../components/Quickstart.vue";
import { QuickActionType } from "../types";
import { isDBA, isDeveloper, isOwner } from "../utils";

interface LocalState {
  showMobileOverlay: boolean;
  showTrialModal: boolean;
  showReleaseModal: boolean;
}

const actuatorStore = useActuatorV1Store();
const subscriptionStore = useSubscriptionV1Store();
const route = useRoute();
const router = useRouter();

const state = reactive<LocalState>({
  showMobileOverlay: false,
  showTrialModal: false,
  showReleaseModal: false,
});

const hasPermission = hasWorkspacePermissionV1(
  "bb.permission.workspace.manage-subscription",
  useCurrentUserV1().value.userRole
);

const { isDemo } = storeToRefs(actuatorStore);

actuatorStore.tryToRemindRelease().then((openRemindModal) => {
  state.showReleaseModal = openRemindModal;
});

const canUpgrade = computed(() => {
  return actuatorStore.hasNewRelease;
});

const currentUserV1 = useCurrentUserV1();

const quickActionList = computed(() => {
  const role = currentUserV1.value.userRole;
  const quickActionListFunc =
    router.currentRoute.value.meta.quickActionListByRole;
  const listByRole = quickActionListFunc
    ? quickActionListFunc(router.currentRoute.value)
    : new Map();
  const list: QuickActionType[] = [];

  // We write this way because for free version, the user wears the three role hat,
  // and we want to display all quick actions relevant to those three roles without duplication.
  if (isOwner(role)) {
    for (const item of listByRole.get("OWNER") || []) {
      list.push(item);
    }
  }

  if (isDBA(role)) {
    for (const item of listByRole.get("DBA") || []) {
      if (
        !list.find((myItem: QuickActionType) => {
          return item == myItem;
        })
      ) {
        list.push(item);
      }
    }
  }

  if (isDeveloper(role)) {
    for (const item of listByRole.get("DEVELOPER") || []) {
      if (
        !list.find((myItem: QuickActionType) => {
          return item == myItem;
        })
      ) {
        list.push(item);
      }
    }
  }
  return list;
});

const showBreadcrumb = computed(() => {
  const name = router.currentRoute.value.name;
  return !(name === "workspace.home" || name === "workspace.profile");
});

const version = computed(() => {
  const v = actuatorStore.version;
  if (v.split(".").length == 3) {
    return `v${v}`;
  }
  return v;
});

const gitCommit = computed(() => {
  return `${actuatorStore.gitCommit.substring(0, 7)}`;
});

const currentPlan = computed((): string => {
  const plan = subscriptionStore.currentPlan;
  switch (plan) {
    case PlanType.TEAM:
      return "subscription.plan.team.title";
    case PlanType.ENTERPRISE:
      return "subscription.plan.enterprise.title";
    default:
      if (hasPermission) {
        return "subscription.plan.try";
      }
      return "subscription.plan.free.title";
  }
});

const isFreePlan = computed((): boolean => {
  const plan = subscriptionStore.currentPlan;
  return plan === PlanType.FREE;
});
</script>
