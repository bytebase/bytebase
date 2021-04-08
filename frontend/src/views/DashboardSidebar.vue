<template>
  <!-- Navigation -->
  <nav class="px-2 space-y-2">
    <div class="space-y-1">
      <router-link
        to="/"
        class="outline-item group flex items-center px-2 py-2"
      >
        <svg
          class="w-5 h-5 mr-2"
          fill="currentColor"
          viewBox="0 0 20 20"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            d="M10.707 2.293a1 1 0 00-1.414 0l-7 7a1 1 0 001.414 1.414L4 10.414V17a1 1 0 001 1h2a1 1 0 001-1v-2a1 1 0 011-1h2a1 1 0 011 1v2a1 1 0 001 1h2a1 1 0 001-1v-6.586l.293.293a1 1 0 001.414-1.414l-7-7z"
          ></path>
        </svg>
        Home
      </router-link>
      <router-link
        to="/inbox"
        class="outline-item group flex items-center px-2 py-2"
      >
        <svg
          class="w-5 h-5 mr-2"
          fill="currentColor"
          viewBox="0 0 20 20"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            fill-rule="evenodd"
            d="M5 3a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2V5a2 2 0 00-2-2H5zm0 2h10v7h-2l-1 2H8l-1-2H5V5z"
            clip-rule="evenodd"
          ></path>
        </svg>
        Inbox
      </router-link>
    </div>
    <div>
      <BookmarkListSidePanel />
    </div>
    <div class="mt-4">
      <DatabaseListSidePanel :mode="'Owner'" />
    </div>
    <div class="mt-4">
      <DatabaseListSidePanel :mode="'Grant'" />
    </div>
  </nav>
</template>

<script lang="ts">
import { reactive } from "vue";
import BookmarkListSidePanel from "../components/BookmarkListSidePanel.vue";
import DatabaseListSidePanel from "../components/DatabaseListSidePanel.vue";

interface LocalState {
  hasUnreadMessage: boolean;
}

export default {
  name: "DashboardSidebar",
  props: {},
  components: {
    BookmarkListSidePanel,
    DatabaseListSidePanel,
  },
  setup(props, ctx) {
    const state = reactive<LocalState>({
      hasUnreadMessage: Math.random() > 0.5,
    });

    return {
      state,
    };
  },
};
</script>
