<template>
  <div class="h-full flex overflow-hidden">
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
            class="outline-item group flex items-center px-4 pt-4 pb-2"
          >
            <heroicons-outline:archive class="w-5 h-5 mr-2" />
            {{ $t("common.archive") }}
          </router-link>
          <div
            class="flex-shrink-0 flex border-t border-block-border px-3 py-2"
          >
            <router-link
              to="/setting/subscription"
              exact-active-class
              class="text-sm text-accent flex"
              :class="isFreePlan ? 'text-accent' : ''"
            >
              <heroicons-solid:sparkles class="w-5 h-5" />
              {{ $t(currentPlan) }}
            </router-link>
            <div class="text-sm ml-auto text-control-light">{{ version }}</div>
          </div>
        </div>
        <div class="flex-shrink-0 w-14" aria-hidden="true">
          <!-- Force sidebar to shrink to fit close icon -->
        </div>
      </div>
    </div>

    <!-- Static sidebar for desktop -->
    <aside class="hidden md:flex md:flex-shrink-0">
      <div class="flex flex-col w-52">
        <!-- Sidebar component, swap this element with another sidebar if you like -->
        <div class="flex-1 flex flex-col py-2 overflow-y-auto">
          <router-view name="leftSidebar" />
        </div>
        <router-link
          to="/archive"
          class="outline-item group flex items-center px-4 pt-4 pb-2"
        >
          <heroicons-outline:archive class="w-5 h-5 mr-2" />
          {{ $t("common.archive") }}
        </router-link>
        <div
          v-if="showQuickstart"
          class="flex-shrink-0 flex justify-center border-t border-block-border py-2"
        >
          <Quickstart />
        </div>
        <div class="flex-shrink-0 flex border-t border-block-border px-3 py-2">
          <router-link
            to="/setting/subscription"
            exact-active-class
            class="text-sm flex"
            :class="isFreePlan ? 'text-accent' : ''"
          >
            <heroicons-outline:sparkles class="w-5 h-5" />
            {{ $t(currentPlan) }}
          </router-link>
          <div class="text-sm ml-auto text-control-light">
            {{ version }}
          </div>
        </div>
      </div>
    </aside>
    <div
      class="flex flex-col min-w-0 flex-1 border-l border-r border-block-border"
    >
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
          <IntroBanner v-if="showIntro" />
          <div v-if="quickActionList.length > 0" class="mx-4 mt-4">
            <QuickActionPanel :quick-action-list="quickActionList" />
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
</template>

<script lang="ts">
import { computed, defineComponent, reactive } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import Breadcrumb from "../components/Breadcrumb.vue";
import IntroBanner from "../components/IntroBanner.vue";
import Quickstart from "../components/Quickstart.vue";
import QuickActionPanel from "../components/QuickActionPanel.vue";
import { QuickActionType } from "../types";
import { isDBA, isDeveloper, isOwner } from "../utils";
import { PlanType } from "../types";
import { useActuatorStore, useSubscriptionStore } from "@/store";

interface LocalState {
  showMobileOverlay: boolean;
}

export default defineComponent({
  name: "BodyLayout",
  components: {
    Breadcrumb,
    IntroBanner,
    Quickstart,
    QuickActionPanel,
  },
  setup() {
    const store = useStore();
    const actuatorStore = useActuatorStore();
    const subscriptionStore = useSubscriptionStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      showMobileOverlay: false,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const quickActionList = computed(() => {
      const role = currentUser.value.role;
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

    const showIntro = computed(() => {
      return !store.getters["uistate/introStateByKey"]("general.overview");
    });

    const showQuickstart = computed(() => {
      // Do not show quickstart in demo mode since we don't expect user to alter the data
      return (
        !actuatorStore.isDemo &&
        !store.getters["uistate/introStateByKey"]("hidden")
      );
    });

    const version = computed(() => {
      const v = actuatorStore.version;
      if (v.split(".").length == 3) {
        return `v${v}`;
      }
      return v;
    });

    const currentPlan = computed((): string => {
      const plan = subscriptionStore.currentPlan;
      switch (plan) {
        case PlanType.TEAM:
          return "subscription.plan.team.title";
        case PlanType.ENTERPRISE:
          return "subscription.plan.enterprise.title";
        default:
          return "subscription.plan.try";
      }
    });

    const isFreePlan = computed((): boolean => {
      const plan = subscriptionStore.currentPlan;
      return plan === PlanType.FREE;
    });

    return {
      state,
      quickActionList,
      showBreadcrumb,
      showIntro,
      showQuickstart,
      version,
      currentPlan,
      isFreePlan,
    };
  },
});
</script>
