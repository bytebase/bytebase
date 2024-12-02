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
import type { Action } from "@bytebase/vue-kbar";
import { defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import CommonSidebar from "@/components/CommonSidebar.vue";
import { useGlobalDatabaseActions } from "@/components/KBar/useDatabaseActions";
import { useProjectActions } from "@/components/KBar/useProjectActions";
import { WORKSPACE_ROUTE_LANDING } from "@/router/dashboard/workspaceRoutes";
import { SQL_EDITOR_HOME_MODULE } from "@/router/sqlEditor";
import { useAppFeature } from "@/store";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";
import { useDashboardSidebar, type DashboardSidebarItem } from "@/utils";

const { t } = useI18n();
const router = useRouter();
const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");
const { getItemClass, dashboardSidebarItemList } = useDashboardSidebar();

const logoRedirect = computed(() => {
  if (databaseChangeMode.value === DatabaseChangeMode.EDITOR) {
    return SQL_EDITOR_HOME_MODULE;
  }
  return WORKSPACE_ROUTE_LANDING;
});

const navigationKbarActions = computed((): Action[] => {
  return dashboardSidebarItemList.value
    .reduce((list, item) => {
      if (!item.children || item.children.length === 0) {
        if (item.navigationId && item.name && !item.hide) {
          list.push(item);
        }
      } else {
        for (const child of item.children) {
          if (child.navigationId && child.name && !child.hide) {
            list.push(child);
          }
        }
      }
      return list;
    }, [] as DashboardSidebarItem[])
    .map((item) => {
      return defineAction({
        id: item.navigationId,
        name: item.title,
        section: t("kbar.navigation"),
        shortcut: item.shortcuts,
        keywords: item.title?.toLocaleLowerCase(),
        perform: () => router.push({ name: item.name }),
      });
    });
});
useRegisterActions(navigationKbarActions);

useProjectActions(10);
useGlobalDatabaseActions(10);
</script>
