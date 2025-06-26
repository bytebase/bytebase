<template>
  <!-- Navigation -->
  <CommonSidebar
    :key="'dashboard'"
    :item-list="dashboardSidebarItemList"
    :get-item-class="getItemClass"
    :logo-redirect="logoRedirect"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import CommonSidebar from "@/components/v2/Sidebar/CommonSidebar.vue";
import { WORKSPACE_ROUTE_LANDING } from "@/router/dashboard/workspaceRoutes";
import { SQL_EDITOR_HOME_MODULE } from "@/router/sqlEditor";
import { useAppFeature } from "@/store";
import { DatabaseChangeMode } from "@/types/proto-es/v1/setting_service_pb";
import { useDashboardSidebar } from "@/utils";

const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");
const { getItemClass, dashboardSidebarItemList } = useDashboardSidebar();

const logoRedirect = computed(() => {
  if (databaseChangeMode.value === DatabaseChangeMode.EDITOR) {
    return SQL_EDITOR_HOME_MODULE;
  }
  return WORKSPACE_ROUTE_LANDING;
});
</script>
