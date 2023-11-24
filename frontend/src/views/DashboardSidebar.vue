<template>
  <!-- Navigation -->
  <nav class="overflow-y-hidden flex flex-col">
    <BytebaseLogo class="w-full px-4 shrink-0" />

    <div class="flex-1 overflow-y-auto px-2">
      <router-link to="/" class="outline-item group flex items-center">
        <div
          class="outline-item group flex items-center px-2 py-1.5 capitalize"
          data-label="bb-dashboard-sidebar-home-button"
        >
          <HomeIcon class="w-5 h-5 mr-2" />
          {{ $t("issue.my-issues") }}
        </div>
      </router-link>

      <router-link to="/project" class="outline-item group flex items-center">
        <div
          class="outline-item group flex items-center px-2 py-1.5 capitalize"
          data-label="bb-dashboard-sidebar-home-button"
        >
          <GalleryHorizontalEndIcon class="w-5 h-5 mr-2" />
          {{ $t("common.projects") }}
        </div>
      </router-link>

      <router-link
        v-if="shouldShowInstanceEntry"
        to="/instance"
        class="outline-item group flex items-center"
      >
        <div
          class="outline-item group flex items-center px-2 py-1.5 capitalize"
          data-label="bb-dashboard-sidebar-home-button"
        >
          <LayersIcon class="w-5 h-5 mr-2" />
          {{ $t("common.instances") }}
        </div>
      </router-link>

      <router-link to="/db" class="outline-item group flex items-center">
        <div
          class="outline-item group flex items-center px-2 py-1.5 capitalize"
          data-label="bb-dashboard-sidebar-home-button"
        >
          <DatabaseIcon class="w-5 h-5 mr-2" />
          {{ $t("common.databases") }}
        </div>
      </router-link>

      <router-link
        to="/environment"
        class="outline-item group flex items-center"
      >
        <div
          class="outline-item group flex items-center px-2 py-1.5 capitalize"
          data-label="bb-dashboard-sidebar-home-button"
        >
          <SquareStackIcon class="w-5 h-5 mr-2" />
          {{ $t("common.environments") }}
        </div>
      </router-link>

      <router-link
        v-if="shouldShowSyncSchemaEntry"
        to="/sync-schema"
        class="outline-item group flex items-center px-2 py-1.5 capitalize"
      >
        <RefreshCcwIcon class="w-5 h-5 mr-2" />
        {{ $t("database.sync-schema.title") }}
      </router-link>
      <router-link
        to="/slow-query"
        class="outline-item group flex items-center px-2 py-1.5 capitalize"
      >
        <TurtleIcon class="w-5 h-auto mr-2" />
        {{ $t("slow-query.slow-queries") }}
      </router-link>
      <router-link
        to="/export-center"
        class="outline-item group flex items-center px-2 py-1.5 capitalize"
      >
        <DownloadIcon class="w-5 h-5 mr-2" />
        {{ $t("export-center.self") }}
      </router-link>
      <router-link
        to="/anomaly-center"
        class="outline-item group flex items-center px-2 py-1.5 capitalize"
      >
        <ShieldAlertIcon class="w-5 h-5 mr-2" />
        {{ $t("anomaly-center") }}
      </router-link>
    </div>
  </nav>
</template>

<script lang="ts" setup>
import {
  HomeIcon,
  DatabaseIcon,
  TurtleIcon,
  RefreshCcwIcon,
  DownloadIcon,
  ShieldAlertIcon,
  GalleryHorizontalEndIcon,
  LayersIcon,
  SquareStackIcon,
} from "lucide-vue-next";
import { computed } from "vue";
import { RouterLink } from "vue-router";
import BytebaseLogo from "@/components/BytebaseLogo.vue";
import {
  useCurrentUserV1,
  useCurrentUserIamPolicy,
  useProjectV1ListByCurrentUser,
} from "@/store";
import { hasWorkspacePermissionV1 } from "../utils";

const currentUserV1 = useCurrentUserV1();

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

const shouldShowInstanceEntry = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-instance",
    currentUserV1.value.userRole
  );
});
</script>
