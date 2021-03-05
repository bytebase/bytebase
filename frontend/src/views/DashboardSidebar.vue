<template>
  <!-- Navigation -->
  <nav class="px-2 space-y-4">
    <div class="space-y-1">
      <router-link
        to="/"
        class="outline-item group flex items-center px-2 py-2"
        active-class="outline-item"
      >
        <svg
          class="mr-3 h-6 w-6 text-control-light group-hover:text-control-light group-focus:text-control-light-hover"
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
    </div>
    <!-- <div class="mt-4">
      <ProjectListSidePanel />
    </div>
    <div class="mt-4">
      <GroupListSidePanel />
    </div> -->
  </nav>
</template>

<script lang="ts">
import { onMounted, reactive } from "vue";
import { useStore } from "vuex";
import BookmarkListSidePanel from "../components/BookmarkListSidePanel.vue";
// import GroupListSidePanel from "../components/GroupListSidePanel.vue";
// import ProjectListSidePanel from "../components/ProjectListSidePanel.vue";

interface LocalState {
  hasUnreadMessage: boolean;
}

export default {
  name: "DashboardSidebar",
  props: {},
  components: {
    BookmarkListSidePanel,
    // GroupListSidePanel,
    // ProjectListSidePanel,
  },
  setup(props, ctx) {
    const store = useStore();

    const state = reactive<LocalState>({
      hasUnreadMessage: Math.random() > 0.5,
    });

    const restoreExpandState = () => {
      store.dispatch("uistate/restoreExpandState").catch((error) => {
        console.log(error);
      });
    };

    onMounted(restoreExpandState);

    return {
      state,
    };
  },
};
</script>
