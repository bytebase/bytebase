<template>
  <!-- Navigation -->
  <nav class="px-2">
    <router-link to="/" class="outline-item group flex items-center px-2 py-2">
      <heroicons-outline:home class="w-5 h-5 mr-2" />
      {{ $t("common.home") }}
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
</script>
