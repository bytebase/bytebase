<template>
  <!-- Navigation -->
  <nav class="px-2">
    <router-link to="/" class="outline-item group flex items-center px-2 py-2">
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
        <template v-if="hasUnreadMessage">
          <path
            d="M8.707 7.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l2-2a1 1 0 00-1.414-1.414L11 7.586V3a1 1 0 10-2 0v4.586l-.293-.293z"
          ></path>
          <path
            d="M3 5a2 2 0 012-2h1a1 1 0 010 2H5v7h2l1 2h4l1-2h2V5h-1a1 1 0 110-2h1a2 2 0 012 2v10a2 2 0 01-2 2H5a2 2 0 01-2-2V5z"
          ></path>
        </template>
        <path
          v-else
          fill-rule="evenodd"
          d="M5 3a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2V5a2 2 0 00-2-2H5zm0 2h10v7h-2l-1 2H8l-1-2H5V5z"
          clip-rule="evenodd"
        ></path>
      </svg>
      Inbox
    </router-link>
    <div>
      <BookmarkListSidePanel />
    </div>
    <div class="mt-1">
      <DatabaseListSidePanel :mode="'Owner'" />
    </div>
    <div class="mt-1">
      <DatabaseListSidePanel :mode="'Grant'" />
    </div>
  </nav>
</template>

<script lang="ts">
import { computed } from "vue";
import { useStore } from "vuex";
import BookmarkListSidePanel from "../components/BookmarkListSidePanel.vue";
import DatabaseListSidePanel from "../components/DatabaseListSidePanel.vue";
import { Message } from "../types";

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
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const hasUnreadMessage = computed(() => {
      const list = store.getters["message/messageListByUser"](
        currentUser.value.id
      );
      return list.find((item: Message) => {
        return item.status == "DELIVERED";
      });
    });

    return {
      hasUnreadMessage,
    };
  },
};
</script>
