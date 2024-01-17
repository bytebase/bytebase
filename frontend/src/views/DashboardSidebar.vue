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
import {
  useCurrentUserV1,
  useCurrentUserIamPolicy,
  useProjectV1ListByCurrentUser,
} from "@/store";
import { hasWorkspacePermissionV2 } from "../utils";

const { t } = useI18n();
const currentUserV1 = useCurrentUserV1();
const route = useRoute();
const router = useRouter();

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
  return hasWorkspacePermissionV2(currentUserV1.value, "bb.instances.list");
});

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

const dashboardSidebarItemList = computed((): SidebarItem[] => {
  return [
    {
      title: t("issue.my-issues"),
      icon: h(HomeIcon),
      name: WORKSPACE_HOME_MODULE,
      type: "route",
    },
    {
      title: t("common.projects"),
      icon: h(GalleryHorizontalEndIcon),
      name: PROJECT_V1_ROUTE_DASHBOARD,
      type: "route",
    },
    {
      title: t("common.instances"),
      icon: h(LayersIcon),
      name: INSTANCE_ROUTE_DASHBOARD,
      type: "route",
      hide: !shouldShowInstanceEntry.value,
    },
    {
      title: t("common.databases"),
      icon: h(DatabaseIcon),
      name: DATABASE_ROUTE_DASHBOARD,
      type: "route",
    },
    {
      title: t("common.environments"),
      icon: h(SquareStackIcon),
      name: ENVIRONMENT_V1_ROUTE_DASHBOARD,
      type: "route",
    },
    {
      type: "divider",
    },
    {
      title: t("slow-query.slow-queries"),
      icon: h(TurtleIcon),
      name: WORKSPACE_ROUTE_SLOW_QUERY,
      type: "route",
      hide: !shouldShowSyncSchemaEntry.value,
    },
    {
      title: t("export-center.self"),
      icon: h(DownloadIcon),
      name: WORKSPACE_ROUTE_EXPORT_CENTER,
      type: "route",
    },
    {
      title: t("anomaly-center"),
      icon: h(ShieldAlertIcon),
      name: WORKSPACE_ROUTE_ANOMALY_CENTER,
      type: "route",
    },
  ];
});

const navigationKbarActions = computed(() => {
  const actions: Action[] = [];
  actions.push(
    defineAction({
      id: "bb.navigation.projects",
      name: "Projects",
      shortcut: ["g", "p"],
      section: t("kbar.navigation"),
      keywords: "navigation",
      perform: () => router.push({ name: PROJECT_V1_ROUTE_DASHBOARD }),
    }),
    defineAction({
      id: "bb.navigation.databases",
      name: "Databases",
      shortcut: ["g", "d"],
      section: t("kbar.navigation"),
      keywords: "navigation db",
      perform: () => router.push({ name: DATABASE_ROUTE_DASHBOARD }),
    })
  );

  if (shouldShowInstanceEntry.value) {
    actions.push(
      defineAction({
        id: "bb.navigation.instances",
        name: "Instances",
        shortcut: ["g", "i"],
        section: t("kbar.navigation"),
        keywords: "navigation",
        perform: () => router.push({ name: INSTANCE_ROUTE_DASHBOARD }),
      })
    );
  }
  actions.push(
    defineAction({
      id: "bb.navigation.environments",
      name: "Environments",
      shortcut: ["g", "e"],
      section: t("kbar.navigation"),
      keywords: "navigation",
      perform: () => router.push({ name: ENVIRONMENT_V1_ROUTE_DASHBOARD }),
    })
  );
  actions.push(
    defineAction({
      id: "bb.navigation.slow-query",
      name: "Slow Query",
      section: t("kbar.navigation"),
      shortcut: ["g", "s", "q"],
      keywords: "slow query",
      perform: () => router.push({ name: WORKSPACE_ROUTE_SLOW_QUERY }),
    }),
    defineAction({
      id: "bb.navigation.export-center",
      name: "Export Center",
      section: t("kbar.navigation"),
      shortcut: ["g", "x", "c"],
      keywords: "export center",
      perform: () => router.push({ name: WORKSPACE_ROUTE_EXPORT_CENTER }),
    }),
    defineAction({
      id: "bb.navigation.anomaly-center",
      name: "Anomaly Center",
      shortcut: ["g", "a", "c"],
      section: t("kbar.navigation"),
      keywords: "anomaly center",
      perform: () => router.push({ name: WORKSPACE_ROUTE_ANOMALY_CENTER }),
    })
  );
  return actions;
});
useRegisterActions(navigationKbarActions);

useProjectActions();
useGlobalDatabaseActions();
</script>
