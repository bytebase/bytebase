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
  useCurrentUserV1,
  useCurrentUserIamPolicy,
  useProjectV1ListByCurrentUser,
} from "@/store";
import { hasWorkspacePermissionV1 } from "../utils";

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
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-instance",
    currentUserV1.value.userRole
  );
});

const getItemClass = (path: string | undefined): string[] => {
  const { path: current } = route;
  const isActiveRoute = path === current || current.startsWith(`${path}/`);
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
      path: "/",
      type: "route",
    },
    {
      title: t("common.projects"),
      icon: h(GalleryHorizontalEndIcon),
      path: "/project",
      type: "route",
    },
    {
      title: t("common.instances"),
      icon: h(LayersIcon),
      path: "/instance",
      type: "route",
      hide: !shouldShowInstanceEntry.value,
    },
    {
      title: t("common.databases"),
      icon: h(DatabaseIcon),
      path: "/db",
      type: "route",
    },
    {
      title: t("common.environments"),
      icon: h(SquareStackIcon),
      path: "/environment",
      type: "route",
    },
    {
      type: "divider",
    },
    {
      title: t("slow-query.slow-queries"),
      icon: h(TurtleIcon),
      path: "/slow-query",
      type: "route",
      hide: !shouldShowSyncSchemaEntry.value,
    },
    {
      title: t("export-center.self"),
      icon: h(DownloadIcon),
      path: "/export-center",
      type: "route",
    },
    {
      title: t("anomaly-center"),
      icon: h(ShieldAlertIcon),
      path: "/anomaly-center",
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
      perform: () => router.push({ name: "workspace.project" }),
    }),
    defineAction({
      id: "bb.navigation.databases",
      name: "Databases",
      shortcut: ["g", "d"],
      section: t("kbar.navigation"),
      keywords: "navigation db",
      perform: () => router.push({ name: "workspace.database" }),
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
        perform: () => router.push({ name: "workspace.instance" }),
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
      perform: () => router.push({ name: "workspace.environment" }),
    })
  );
  if (shouldShowSyncSchemaEntry.value) {
    actions.push(
      defineAction({
        id: "bb.navigation.sync-schema",
        name: "Sync Schema",
        shortcut: ["g", "s", "s"],
        section: t("kbar.navigation"),
        keywords: "sync schema",
        perform: () => router.push({ name: "workspace.sync-schema" }),
      })
    );
  }
  actions.push(
    defineAction({
      id: "bb.navigation.slow-query",
      name: "Slow Query",
      section: t("kbar.navigation"),
      shortcut: ["g", "s", "q"],
      keywords: "slow query",
      perform: () => router.push({ name: "workspace.slow-query" }),
    }),
    defineAction({
      id: "bb.navigation.export-center",
      name: "Export Center",
      section: t("kbar.navigation"),
      shortcut: ["g", "x", "c"],
      keywords: "export center",
      perform: () => router.push({ name: "workspace.export-center" }),
    }),
    defineAction({
      id: "bb.navigation.anomaly-center",
      name: "Anomaly Center",
      shortcut: ["g", "a", "c"],
      section: t("kbar.navigation"),
      keywords: "anomaly center",
      perform: () => router.push({ name: "workspace.anomaly-center" }),
    })
  );
  return actions;
});
useRegisterActions(navigationKbarActions);

useProjectActions();
useGlobalDatabaseActions();
</script>
