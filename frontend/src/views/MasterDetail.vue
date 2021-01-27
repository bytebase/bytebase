<template>
  <!--
  Tailwind UI components require Tailwind CSS v1.8 and the @tailwindcss/ui plugin.
  Read the documentation to get started: https://tailwindui.com/documentation
-->
  <!-- Background color split screen for large screens -->
  <div class="relative min-h-screen flex flex-col">
    <div class="flex-grow w-full max-w-full mx-auto lg:flex">
      <!-- Off-canvas menu for mobile, show/hide based on off-canvas menu state. -->
      <div v-if="showMobileOverlay" class="lg:hidden">
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
          <div class="sidebar relative flex-1 flex flex-col max-w-xs pt-4">
            <div class="absolute top-0 right-0 -mr-14 p-1">
              <button
                @click.prevent="state.showMobileOverlay = false"
                class="flex items-center justify-center h-12 w-12 rounded-full focus:outline-none focus:bg-gray-600"
                aria-label="Close sidebar"
              >
                <svg
                  class="h-6 w-6 text-white"
                  stroke="currentColor"
                  fill="none"
                  viewBox="0 0 24 24"
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
            <div class="flex-shrink-0 flex justify-between px-4 mb-6">
              <router-link to="/" active-class="" exact-active-class=""
                ><img
                  class="h-10 w-auto"
                  src="../assets/logo.svg"
                  alt="Bytebase"
              /></router-link>
              <ProfileDropdown />
            </div>
            <router-view name="leftSidebar" />
            <div
              class="flex-shrink-0 flex flex-row border-t border-block-border p-3 text-gray-700"
            >
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
          </div>
        </div>
      </div>
      <div
        class="bg-normal hidden lg:flex lg:flex-shrink-0 flex-col w-64 border-r border-block-border pt-4"
      >
        <div class="flex justify-between flex-shrink-0 px-4 mb-2">
          <router-link to="/" active-class="" exact-active-class=""
            ><img class="h-10 w-auto" src="../assets/logo.svg" alt="Bytebase"
          /></router-link>
          <div class="flex items-center flex-shrink-0 pl-4">
            <ProfileDropdown />
          </div>
        </div>
        <router-view name="leftSidebar" />
        <div
          class="flex-shrink-0 flex flex-row border-t border-block-border p-3 text-gray-700"
        >
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
      </div>
      <div class="lg:flex flex-col flex-1">
        <div
          class="bg-normal flex-shrink-0 border-b border-block-border px-2 py-2"
        >
          <TheHeader v-on:open-sidebar="state.showMobileOverlay = true" />
        </div>
        <div class="lg:flex flex-1">
          <!-- Main Content-->
          <div class="bg-normal lg:min-w-0 lg:flex-1">
            <router-view name="content" />
          </div>
          <!-- Activity feed -->
          <div
            class="lg:block bg-gray-50 pr-4 sm:pr-6 lg:pr-8 lg:flex-shrink-0 lg:border-l lg:border-block-border xl:pr-0"
          >
            <div class="pl-6 lg:w-80">
              <router-view name="rightsidebar" />
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive } from "vue";
import ActivitySidebar from "../views/ActivitySidebar.vue";
import ProfileDropdown from "../components/ProfileDropdown.vue";
import TheHeader from "../components/TheHeader.vue";

interface LocalState {
  showMobileOverlay: boolean;
}

export default {
  name: "MaseterDetail",
  components: {
    ActivitySidebar,
    ProfileDropdown,
    TheHeader,
  },
  setup(props, ctx) {
    const state = reactive<LocalState>({
      showMobileOverlay: false,
    });

    return state;
  },
};
</script>
