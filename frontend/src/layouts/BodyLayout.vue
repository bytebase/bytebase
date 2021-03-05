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
              @click.prevent="state.showMobileOverlay = false"
              type="button"
              class="ml-1 flex items-center justify-center h-10 w-10 rounded-full focus:outline-none focus:ring-2 focus:ring-inset focus:ring-white"
            >
              <span class="sr-only">Close sidebar</span>
              <!-- Heroicon name: x -->
              <svg
                class="h-6 w-6 text-white"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-hidden="true"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>
          <!-- Mobile Sidebar -->
          <div class="flex-1 h-0 py-4 overflow-y-auto">
            <router-view name="leftSidebar" />
          </div>
          <div class="flex-shrink-0 flex border-t border-gray-200 p-4">
            <a href="#" class="flex-shrink-0 group block">
              <div class="flex items-center">
                <svg
                  class="w-5 h-5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  ></path>
                </svg>
                <a href="/" class="ml-1 text-sm">Help &amp; Feedback</a>
              </div>
            </a>
          </div>
        </div>
        <div class="flex-shrink-0 w-14" aria-hidden="true">
          <!-- Force sidebar to shrink to fit close icon -->
        </div>
      </div>
    </div>

    <!-- Static sidebar for desktop -->
    <aside class="hidden md:flex md:flex-shrink-0">
      <div class="flex flex-col w-48">
        <!-- Sidebar component, swap this element with another sidebar if you like -->
        <div class="flex-1 flex flex-col py-4 overflow-y-auto">
          <router-view name="leftSidebar" />
        </div>
        <div class="flex-shrink-0 flex border-t border-gray-200 p-4">
          <a href="#" class="flex-shrink-0 w-full group block">
            <div class="flex items-center">
              <svg
                class="w-5 h-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                ></path>
              </svg>
              <a href="/" class="ml-1 text-sm">Help &amp; Feedback</a>
            </div>
          </a>
        </div>
      </div>
    </aside>
    <div
      class="flex flex-col min-w-0 flex-1 border-l border-r border-block-border"
    >
      <!-- Static sidebar for mobile -->
      <aside class="md:hidden">
        <div
          class="flex items-center justify-start bg-gray-50 border-b border-gray-200 px-4 py-1.5"
        >
          <div>
            <button
              @click.prevent="state.showMobileOverlay = true"
              type="button"
              class="-mr-3 h-12 w-12 inline-flex items-center justify-center rounded-md text-gray-500 hover:text-gray-900"
            >
              <span class="sr-only">Open sidebar</span>
              <!-- Heroicon name: menu -->
              <svg
                class="h-6 w-6"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-hidden="true"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M4 6h16M4 12h16M4 18h16"
                />
              </svg>
            </button>
          </div>
          <div v-if="!isHome" class="ml-4">
            <Breadcrumb />
          </div>
        </div>
      </aside>
      <div class="w-full mx-auto md:flex">
        <div class="md:min-w-0 md:flex-1">
          <div v-if="!isHome" class="hidden md:block mx-4 mt-4">
            <Breadcrumb />
          </div>
          <div v-if="quickActionList" class="mx-4 mt-4">
            <QuickActionPanel :quickActionList="quickActionList" />
          </div>
        </div>
      </div>
      <!-- This area may scroll -->
      <div class="md:min-w-0 md:flex-1 overflow-y-auto mt-4">
        <!-- Start main area-->
        <router-view name="content" />
        <!-- End main area -->
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import Breadcrumb from "../components/Breadcrumb.vue";
import QuickActionPanel from "../components/QuickActionPanel.vue";

interface LocalState {
  showMobileOverlay: boolean;
}

export default {
  name: "BodyLayout",
  components: {
    Breadcrumb,
    QuickActionPanel,
  },
  setup(props, ctx) {
    const router = useRouter();

    const state = reactive<LocalState>({
      showMobileOverlay: false,
    });

    const isHome = computed(() => {
      return router.currentRoute.value.path == "/";
    });

    const quickActionList = computed(() => {
      return router.currentRoute.value.meta.quickActionList;
    });

    return {
      state,
      isHome,
      quickActionList,
    };
  },
};
</script>
