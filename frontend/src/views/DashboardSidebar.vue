<template>
  <!-- Navigation -->
  <nav class="overflow-y-hidden flex flex-col">
    <BytebaseLogo class="w-full px-4 shrink-0" />

    <div class="flex-1 overflow-y-auto px-2 pt-1">
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
            <span v-else class="tracking-tighter transform scale-x-90"
              >Ctrl</span
            >
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
          {{ $t("issue.my-issues") }}
        </div>
      </router-link>
      <a
        href="/sql-editor"
        target="_blank"
        class="outline-item group flex items-center px-2 py-2"
      >
        <heroicons-solid:terminal class="w-5 h-5 mr-2" />
        {{ $t("sql-editor.self") }}
      </a>
      <router-link
        v-if="shouldShowSyncSchemaEntry"
        to="/sync-schema"
        class="outline-item group flex items-center px-2 py-2 capitalize"
      >
        <heroicons-outline:refresh class="w-5 h-5 mr-2" />
        {{ $t("database.sync-schema.title") }}
      </router-link>
      <router-link
        to="/export-center"
        class="outline-item group flex items-center px-2 py-2 capitalize"
      >
        <heroicons-outline:download class="w-5 h-5 mr-2" />
        {{ $t("export-center.self") }}
      </router-link>
      <router-link
        to="/slow-query"
        class="outline-item group flex items-center px-2 py-2 capitalize"
      >
        <img src="../assets/slow-query.svg" class="w-5 h-auto mr-2" />
        {{ $t("slow-query.slow-queries") }}
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
    </div>
  </nav>
</template>

<script lang="ts" setup>
import { useKBarHandler } from "@bytebase/vue-kbar";
import { computed } from "vue";
import BookmarkListSidePanel from "@/components/BookmarkListSidePanel.vue";
import BytebaseLogo from "@/components/BytebaseLogo.vue";
import DatabaseListSidePanel from "@/components/DatabaseListSidePanel.vue";
import ProjectListSidePanel from "@/components/ProjectListSidePanel.vue";
import {
  useCurrentUserIamPolicy,
  useProjectV1ListByCurrentUser,
} from "@/store";

const isMac = navigator.platform.match(/mac/i);

const handler = useKBarHandler();

// Only show sync schema if the user has permission to alter schema of at least one project.
const shouldShowSyncSchemaEntry = computed(() => {
  const { projectList } = useProjectV1ListByCurrentUser();
  const currentUserIamPolicy = useCurrentUserIamPolicy();
  return projectList.value
    .map((project) => {
      return currentUserIamPolicy.allowToChangeDatabaseOfProject(project.name);
    })
    .includes(true);
});

const onClickSearchButton = () => {
  handler.value.show();
};
</script>
