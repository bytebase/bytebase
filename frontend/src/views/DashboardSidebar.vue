<template>
  <!-- Navigation -->
  <nav class="px-2 space-y-1">
    <div class="space-y-1">
      <router-link
        to="/"
        class="outline-item group flex items-center px-2 py-2"
      >
        <svg
          class="mr-2 h-4 w-4 text-control-light group-hover:text-control-light group-focus:text-control-light-hover"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
          />
        </svg>
        Home
      </router-link>
    </div>
    <div>
      <BookmarkListSidePanel />
      <DatabaseListSidePanel />
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
