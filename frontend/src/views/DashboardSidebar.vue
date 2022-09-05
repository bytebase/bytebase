<template>
  <!-- Navigation -->
  <nav class="px-2">
    <button
      class="mb-2 w-full flex items-center justify-between rounded-md border border-control-border bg-white hover:bg-control-bg-hover pl-2 pr-1 py-0.5 outline-none"
      @click="onClickSearchButton"
    >
      <span class="text-control-placeholder">{{ $t("common.search") }}</span>
      <span class="flex items-center space-x-1">
        <kbd
          class="h-5 flex items-center justify-center bg-black bg-opacity-10 rounded text-sm px-1 text-control overflow-y-hidden"
        >
          <span v-if="isMac" class="text-xl px-0.5">⌘</span>
          <span v-else class="tracking-tighter transform scale-x-90">Ctrl</span>
          <span class="ml-1 mr-0.5">K</span>
        </kbd>
      </span>
    </button>

    <router-link to="/" class="outline-item group flex items-center">
      <div
        class="outline-item group flex items-center px-2 py-2"
        data-label="bb-dashboard-sidebar-home-button"
      >
        <heroicons-outline:home class="w-5 h-5 mr-2" />
        {{ $t("common.home") }}
      </div>
    </router-link>
    <router-link
      to="/sql-editor"
      class="outline-item group flex items-center px-2 py-2"
    >
      <heroicons-outline:terminal class="w-5 h-5 mr-2" />
      {{ $t("sql-editor.self") }}
    </router-link>
    <router-link
      to="/anomaly-center"
      class="outline-item group flex items-center px-2 py-2"
    >
      <heroicons-outline:shield-exclamation class="w-5 h-5 mr-2" />
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

<script lang="ts" setup>
import BookmarkListSidePanel from "../components/BookmarkListSidePanel.vue";
import ProjectListSidePanel from "../components/ProjectListSidePanel.vue";
import DatabaseListSidePanel from "../components/DatabaseListSidePanel.vue";
import { useKBarHandler } from "@bytebase/vue-kbar";

const isMac = navigator.platform.match(/mac/i);

const handler = useKBarHandler();

const onClickSearchButton = () => {
  handler.value.show();
};
</script>
