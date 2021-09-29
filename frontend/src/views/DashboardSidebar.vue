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
      class="outline-item group flex items-center justify-between px-2 py-2"
    >
      <div class="flex">
        <svg
          v-if="inboxSummary.hasUnread"
          class="w-5 h-5 mr-2"
          fill="currentColor"
          viewBox="0 0 20 20"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            d="M8.707 7.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l2-2a1 1 0 00-1.414-1.414L11 7.586V3a1 1 0 10-2 0v4.586l-.293-.293z"
          ></path>
          <path
            d="M3 5a2 2 0 012-2h1a1 1 0 010 2H5v7h2l1 2h4l1-2h2V5h-1a1 1 0 110-2h1a2 2 0 012 2v10a2 2 0 01-2 2H5a2 2 0 01-2-2V5z"
          ></path>
        </svg>
        <svg
          v-else
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
      </div>
      <svg
        v-if="inboxSummary.hasUnreadError"
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
          d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
        ></path>
      </svg>
    </router-link>
    <router-link
      v-if="false"
      to="/control-center"
      class="outline-item group flex items-center px-2 py-2"
    >
      <svg
        class="w-5 h-5 mr-2"
        fill="currentColor"
        viewBox="0 0 20 20"
        xmlns="http://www.w3.org/2000/svg"
      >
        <path
          d="M5 4a1 1 0 00-2 0v7.268a2 2 0 000 3.464V16a1 1 0 102 0v-1.268a2 2 0 000-3.464V4zM11 4a1 1 0 10-2 0v1.268a2 2 0 000 3.464V16a1 1 0 102 0V8.732a2 2 0 000-3.464V4zM16 3a1 1 0 011 1v7.268a2 2 0 010 3.464V16a1 1 0 11-2 0v-1.268a2 2 0 010-3.464V4a1 1 0 011-1z"
        ></path>
      </svg>
      Control Center
    </router-link>
    <div>
      <BookmarkListSidePanel />
    </div>
    <div class="mt-1">
      <ProjectListSidePanel />
    </div>
    <div class="mt-1">
      <DatabaseListSidePanel />
    </div>
  </nav>
</template>

<script lang="ts">
import BookmarkListSidePanel from "../components/BookmarkListSidePanel.vue";
import ProjectListSidePanel from "../components/ProjectListSidePanel.vue";
import DatabaseListSidePanel from "../components/DatabaseListSidePanel.vue";
import { computed, watchEffect } from "@vue/runtime-core";
import { useStore } from "vuex";
import { InboxSummary, UNKNOWN_ID } from "../types";

export default {
  name: "DashboardSidebar",
  props: {},
  components: {
    BookmarkListSidePanel,
    ProjectListSidePanel,
    DatabaseListSidePanel,
  },
  setup(props, ctx) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareInboxSummary = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        store.dispatch("inbox/fetchInboxSummaryByUser", currentUser.value.id);
      }
    };

    watchEffect(prepareInboxSummary);

    const inboxSummary = computed((): InboxSummary => {
      return store.getters["inbox/inboxSummaryByUser"](currentUser.value.id);
    });

    return { inboxSummary };
  },
};
</script>
