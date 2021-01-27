<template>
  <!-- This example requires Tailwind CSS v2.0+ -->
  <div class="h-screen flex overflow-hidden bg-white">
    <!-- Off-canvas menu for mobile, show/hide based on off-canvas menu state. -->
    <div v-if="state.showMobileOverlay" class="lg:hidden">
      <div class="fixed inset-0 flex z-40">
        <!--
        Off-canvas menu overlay, show/hide based on off-canvas menu state.

        Entering: "transition-opacity ease-linear duration-300"
          From: "opacity-0"
          To: "opacity-100"
        Leaving: "transition-opacity ease-linear duration-300"
          From: "opacity-100"
          To: "opacity-0"
      -->
        <div class="fixed inset-0">
          <div class="absolute inset-0 bg-gray-600 opacity-75"></div>
        </div>
        <!--
        Off-canvas menu, show/hide based on off-canvas menu state.

        Entering: "transition ease-in-out duration-300 transform"
          From: "-translate-x-full"
          To: "translate-x-0"
        Leaving: "transition ease-in-out duration-300 transform"
          From: "translate-x-0"
          To: "-translate-x-full"
      -->
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
          <div class="flex-1 h-0 pt-5 pb-4 overflow-y-auto">
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
    <div class="hidden lg:flex lg:flex-shrink-0">
      <div class="flex flex-col w-64">
        <!-- Sidebar component, swap this element with another sidebar if you like -->
        <div
          class="flex flex-col h-0 flex-1 border-r border-gray-200 bg-gray-100"
        >
          <div class="flex-1 flex flex-col pt-5 pb-4 overflow-y-auto">
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
      </div>
    </div>
    <div class="flex flex-col min-w-0 flex-1 overflow-hidden">
      <div class="lg:hidden">
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
          <div class="ml-4">
            <Breadcrumb />
          </div>
        </div>
      </div>
      <div class="flex-grow w-full mx-auto lg:flex">
        <!-- Left sidebar & main wrapper -->
        <div class="flex-1 min-w-0 bg-white lg:flex">
          <div class="bg-white lg:min-w-0 lg:flex-1">
            <div class="hidden lg:block px-4 py-3">
              <Breadcrumb />
            </div>

            <div class="h-full overflow-hidden">
              <!-- Start main area-->
              <router-view name="content" />
              <!-- End main area -->
            </div>
          </div>
        </div>

        <div
          v-if="showRightSidebar"
          class="bg-gray-50 lg:flex-shrink-0 lg:border-l lg:border-gray-200"
        >
          <div class="h-full py-6 lg:w-80">
            <!-- Start right column area -->
            <router-view name="rightSidebar" />
            <!-- End right column area -->
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import Breadcrumb from "../components/Breadcrumb.vue";

interface LocalState {
  showMobileOverlay: boolean;
}

export default {
  name: "BodyLayout",
  components: {
    Breadcrumb,
  },
  setup(props, ctx) {
    const state = reactive<LocalState>({
      showMobileOverlay: false,
    });

    // For now, only the home page needs the right sidebar.
    // So this is easier than creating a dedicate layout.
    const showRightSidebar = computed(() => {
      return useRouter().currentRoute.value.meta.rightSidebar;
    });

    return {
      state,
      showRightSidebar,
    };
  },
};
</script>
