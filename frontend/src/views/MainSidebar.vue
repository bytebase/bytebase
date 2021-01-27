<template>
  <!-- Navigation -->
  <nav class="px-2">
    <div class="space-y-1">
      <router-link
        to="/"
        class="sidebar-link group flex items-center px-2 py-2"
        active-class="sidebar-link"
      >
        <svg
          class="mr-3 h-6 w-6 text-gray-500 group-hover:text-gray-500 group-focus:text-gray-600"
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

      <router-link
        to="/inbox"
        class="sidebar-link group flex items-center px-2 py-2"
      >
        <svg
          class="mr-3 h-6 w-6 text-gray-500 group-hover:text-gray-500 group-focus:text-gray-600"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            v-if="state.hasUnreadMessage"
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M8 4H6a2 2 0 00-2 2v12a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2h-2m-4-1v8m0 0l3-3m-3 3L9 8m-5 5h2.586a1 1 0 01.707.293l2.414 2.414a1 1 0 00.707.293h3.172a1 1 0 00.707-.293l2.414-2.414a1 1 0 01.707-.293H20"
          ></path>
          <path
            v-else
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"
          ></path>
        </svg>
        Inbox
      </router-link>
      <router-link
        to="/environment"
        class="sidebar-link group flex items-center px-2 py-2"
      >
        <svg
          class="mr-3 w-6 h-6 text-gray-500 group-hover:text-gray-500 group-focus:text-gray-600"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"
          ></path>
        </svg>
        Environment
      </router-link>
    </div>
    <div class="mt-4">
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
  name: "MainSidebar",
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
