<template>
  <div class="flex items-center justify-between h-16">
    <div class="flex items-center">
      <div class="flex-shrink-0">
        <router-link to="/" active-class="" exact-active-class=""
          ><img class="h-10 w-auto" src="../assets/logo.svg" alt="Bytebase"
        /></router-link>
      </div>
      <div class="hidden md:block">
        <div class="ml-10 flex items-baseline space-x-4">
          <!-- Current: "bg-gray-900 text-white", Default: "text-gray-300 hover:bg-gray-700 hover:text-white" -->
          <router-link to="/" class="link px-3 py-2 rounded-md"
            >Home</router-link
          >

          <router-link to="/instance" class="link px-3 py-2 rounded-md"
            >Instance</router-link
          >

          <router-link to="/environment" class="link px-3 py-2 rounded-md"
            >Environment</router-link
          >
        </div>
      </div>
    </div>
    <div>
      <div class="flex items-center">
        <router-link to="/inbox" class="icon-link p-1 rounded-full">
          <span class="sr-only">View Inbox</span>
          <!-- Heroicon name: bell -->
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
              d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
            />
          </svg>
        </router-link>

        <div class="ml-2">
          <ProfileDropdown />
        </div>
        <div class="ml-2 -mr-2 flex md:hidden">
          <!-- Mobile menu button -->
          <button
            @click.prevent="state.showMobileMenu = !state.showMobileMenu"
            class="icon-link inline-flex items-center justify-center rounded-md"
          >
            <span class="sr-only">Open main menu</span>
            <!--
              Heroicon name: menu

              Menu open: "hidden", Menu closed: "block"
            -->
            <svg
              class="w-6 h-6"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z"
              ></path>
            </svg>
          </button>
        </div>
      </div>
    </div>
  </div>

  <!--
      Mobile menu, toggle classes based on menu state.

      Open: "block", closed: "hidden"
    -->
  <div v-if="state.showMobileMenu" class="block md:hidden">
    <div class="px-2 pt-2 pb-3 space-y-1 sm:px-3">
      <!-- Current: "bg-gray-900 text-white", Default: "text-gray-300 hover:bg-gray-700 hover:text-white" -->
      <router-link to="/" class="link rounded-md block px-3 py-2"
        >Dashboard</router-link
      >

      <router-link to="/instance" class="link rounded-md block px-3 py-2"
        >Instance</router-link
      >

      <router-link to="/environment" class="link rounded-md block px-3 py-2"
        >Environment</router-link
      >
    </div>
  </div>
</template>

<script lang="ts">
import { reactive } from "vue";
import ProfileDropdown from "../components/ProfileDropdown.vue";

interface LocalState {
  showMobileMenu: boolean;
}

export default {
  name: "DashboardHeader",
  components: { ProfileDropdown },
  setup(props, ctx) {
    const state = reactive<LocalState>({
      showMobileMenu: false,
    });

    return { state };
  },
};
</script>
