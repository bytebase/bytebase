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
      to="/anomaly-center"
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
          d="M10 1.944A11.954 11.954 0 012.166 5C2.056 5.649 2 6.319 2 7c0 5.225 3.34 9.67 8 11.317C14.66 16.67 18 12.225 18 7c0-.682-.057-1.35-.166-2.001A11.954 11.954 0 0110 1.944zM11 14a1 1 0 11-2 0 1 1 0 012 0zm0-7a1 1 0 10-2 0v3a1 1 0 102 0V7z"
          clip-rule="evenodd"
        ></path>
      </svg>
      Anomaly Center
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

export default {
  name: "DashboardSidebar",
  props: {},
  components: {
    BookmarkListSidePanel,
    ProjectListSidePanel,
    DatabaseListSidePanel,
  },
  setup(props, ctx) {
    return {};
  },
};
</script>
