<template>
  <!-- Navigation -->
  <CommonSidebar
    :key="'dashboard'"
    :item-list="dashboardSidebarItemList"
    :get-item-class="getItemClass"
  />
</template>

<script lang="ts" setup>
import { Action, defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import {
  HomeIcon,
  DatabaseIcon,
  TurtleIcon,
  DownloadIcon,
  ShieldAlertIcon,
  GalleryHorizontalEndIcon,
  LayersIcon,
  SquareStackIcon,
} from "lucide-vue-next";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { SidebarItem } from "@/components/CommonSidebar.vue";
import { useGlobalDatabaseActions } from "@/components/KBar/useDatabaseActions";
import { useProjectActions } from "@/components/KBar/useProjectActions";
import {
  DATABASE_ROUTE_DASHBOARD,
  ENVIRONMENT_V1_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DASHBOARD,
  WORKSPACE_HOME_MODULE,
  WORKSPACE_ROUTE_SLOW_QUERY,
  WORKSPACE_ROUTE_EXPORT_CENTER,
  WORKSPACE_ROUTE_ANOMALY_CENTER,
} from "@/router/dashboard/workspaceRoutes";
import { useCurrentUserV1 } from "@/store";
import { hasWorkspacePermissionV2 } from "../utils";

interface DashboardSidebarItem extends SidebarItem {
  navigationId: string;
  shortcuts: string[];
  name: string;
  type: "route" | "divider";
}

const { t } = useI18n();
const currentUserV1 = useCurrentUserV1();
const route = useRoute();
const router = useRouter();

const getItemClass = (item: SidebarItem): string[] => {
  const { name: current } = route;
  const isActiveRoute =
    item.name === current?.toString ||
    current?.toString().startsWith(`${item.name}/`);
  const classes: string[] = [];
  if (isActiveRoute) {
    classes.push("router-link-active", "bg-link-hover");
  }
  return classes;
};

const dashboardSidebarItemList = computed((): DashboardSidebarItem[] => {
  return [
    {
      navigationId: "bb.navigation.home",
      title: t("issue.my-issues"),
      icon: h(HomeIcon),
      name: WORKSPACE_HOME_MODULE,
      type: "route",
      shortcuts: [],
    },
    {
      navigationId: "bb.navigation.projects",
      title: t("common.projects"),
      icon: h(GalleryHorizontalEndIcon),
      name: PROJECT_V1_ROUTE_DASHBOARD,
      type: "route",
      shortcuts: ["g", "p"],
      hide: !hasWorkspacePermissionV2(currentUserV1.value, "bb.projects.list"),
    },
    {
      navigationId: "bb.navigation.instances",
      title: t("common.instances"),
      icon: h(LayersIcon),
      name: INSTANCE_ROUTE_DASHBOARD,
      type: "route",
      shortcuts: ["g", "i"],
      hide: !hasWorkspacePermissionV2(currentUserV1.value, "bb.instances.list"),
    },
    {
      navigationId: "bb.navigation.databases",
      title: t("common.databases"),
      icon: h(DatabaseIcon),
      name: DATABASE_ROUTE_DASHBOARD,
      type: "route",
      shortcuts: ["g", "d"],
    },
    {
      navigationId: "bb.navigation.environments",
      title: t("common.environments"),
      icon: h(SquareStackIcon),
      name: ENVIRONMENT_V1_ROUTE_DASHBOARD,
      type: "route",
      shortcuts: ["g", "e"],
      hide: !hasWorkspacePermissionV2(
        currentUserV1.value,
        "bb.environments.list"
      ),
    },
    {
      navigationId: "",
      type: "divider",
      name: "",
      shortcuts: [],
    },
    {
      navigationId: "bb.navigation.slow-query",
      title: t("slow-query.slow-queries"),
      icon: h(TurtleIcon),
      name: WORKSPACE_ROUTE_SLOW_QUERY,
      type: "route",
      shortcuts: ["g", "s", "q"],
    },
    {
      navigationId: "bb.navigation.export-center",
      title: t("export-center.self"),
      icon: h(DownloadIcon),
      name: WORKSPACE_ROUTE_EXPORT_CENTER,
      type: "route",
      shortcuts: ["g", "x", "c"],
    },
    {
      navigationId: "bb.navigation.anomaly-center",
      title: t("anomaly-center"),
      icon: h(ShieldAlertIcon),
      name: WORKSPACE_ROUTE_ANOMALY_CENTER,
      type: "route",
      shortcuts: ["g", "a", "c"],
    },
  ];
});

const navigationKbarActions = computed((): Action[] => {
  return dashboardSidebarItemList.value
    .filter((item) => item.navigationId && item.name && !item.hide)
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

useProjectActions();
useGlobalDatabaseActions();
</script>
