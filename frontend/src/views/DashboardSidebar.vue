<template>
  <!-- Navigation -->
  <nav class="px-2">
    <router-link to="/" class="outline-item group flex items-center px-2 py-2">
      <svg
        class="w-5 h-5 mr-2"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
        ></path>
      </svg>
      {{ $t("common.home") }}
    </router-link>
    <router-link
      to="/anomaly-center"
      class="outline-item group flex items-center px-2 py-2"
    >
      <svg
        class="w-5 h-5 mr-2"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M20.618 5.984A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016zM12 9v2m0 4h.01"
        ></path>
      </svg>
      {{ $t("anomaly-center") }}
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
  components: {
    BookmarkListSidePanel,
    ProjectListSidePanel,
    DatabaseListSidePanel,
  },
};
</script>
