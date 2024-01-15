<template>
  <CommonSidebar
    :key="'project'"
    :item-list="projectSidebarItemList"
    :get-item-class="getItemClass"
    @select="onSelect"
  />
</template>

<script setup lang="ts">
import { defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { startCase } from "lodash-es";
import {
  Database,
  GitBranch,
  CircleDot,
  Users,
  Link,
  Settings,
  RefreshCcw,
  PencilRuler,
} from "lucide-vue-next";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter, useRoute } from "vue-router";
import { SidebarItem } from "@/components/CommonSidebar.vue";
import {
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_ISSUES,
  PROJECT_V1_ROUTE_CHANGE_HISTORIES,
  PROJECT_V1_ROUTE_SYNC_SCHEMA,
  PROJECT_V1_ROUTE_SLOW_QUERIES,
  PROJECT_V1_ROUTE_ANOMALIES,
  PROJECT_V1_ROUTE_ACTIVITIES,
  PROJECT_V1_ROUTE_GITOPS,
  PROJECT_V1_ROUTE_MEMBERS,
  PROJECT_V1_ROUTE_SETTINGS,
  PROJECT_V1_ROUTE_WEBHOOKS,
  PROJECT_V1_ROUTE_WEBHOOK_CREATE,
  PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
  PROJECT_V1_ROUTE_BRANCHES,
  PROJECT_V1_ROUTE_BRANCH_DETAIL,
  PROJECT_V1_ROUTE_BRANCH_ROLLOUT,
  PROJECT_V1_ROUTE_BRANCH_MERGE,
  PROJECT_V1_ROUTE_BRANCH_REBASE,
  PROJECT_V1_ROUTE_CHANGELISTS,
  PROJECT_V1_ROUTE_CHANGELIST_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_GROUPS,
  PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_GROUP_TABLE_GROUP_DETAIL,
  PROJECT_V1_ROUTE_DEPLOYMENT_CONFIG,
} from "@/router/dashboard/projectV1";
import { useCurrentUserV1 } from "@/store";
import { getProjectName } from "@/store/modules/v1/common";
import { DEFAULT_PROJECT_V1_NAME, ProjectPermission } from "@/types";
import { TenantMode } from "@/types/proto/v1/project_service";
import { hasProjectPermissionV2 } from "@/utils";
import { useProjectDatabaseActions } from "../KBar/useDatabaseActions";
import { useCurrentProject } from "./useCurrentProject";

interface ProjectSidebarItem extends SidebarItem {
  title: string;
  type: "div";
  path?: string;
  hide?: boolean;
  permissions?: ProjectPermission[];
  children?: ProjectSidebarItem[];
}

const props = defineProps<{
  projectId?: string;
  issueSlug?: string;
  databaseSlug?: string;
  changeHistorySlug?: string;
}>();

const route = useRoute();
const { t } = useI18n();
const router = useRouter();

const params = computed(() => {
  return {
    projectId: props.projectId,
    issueSlug: props.issueSlug,
    databaseSlug: props.databaseSlug,
    changeHistorySlug: props.changeHistorySlug,
  };
});

const { project } = useCurrentProject(params);
const currentUser = useCurrentUserV1();

const isDefaultProject = computed((): boolean => {
  return project.value.name === DEFAULT_PROJECT_V1_NAME;
});

const isTenantProject = computed((): boolean => {
  return project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED;
});

const filterprojectSidebarByPermissions = (
  sidebarList: ProjectSidebarItem[]
): ProjectSidebarItem[] => {
  return sidebarList
    .filter((item) => {
      return (
        hasProjectPermissionV2(
          project.value,
          currentUser.value,
          "bb.projects.get"
        ) &&
        (item.permissions ?? []).every((permission) =>
          hasProjectPermissionV2(project.value, currentUser.value, permission)
        )
      );
    })
    .map((item) => ({
      ...item,
      children: filterprojectSidebarByPermissions(item.children ?? []),
    }));
};

const projectSidebarItemList = computed((): ProjectSidebarItem[] => {
  const sidebarList: ProjectSidebarItem[] = [
    {
      title: t("common.database"),
      icon: h(Database),
      type: "div",
      children: [
        {
          title: t("common.databases"),
          path: PROJECT_V1_ROUTE_DATABASES,
          type: "div",
          permissions: ["bb.databases.list"],
        },
        {
          title: t("common.groups"),
          path: PROJECT_V1_ROUTE_DATABASE_GROUPS,
          type: "div",
          hide: !isTenantProject.value,
        },
        {
          title: t("common.deployment-config"),
          path: PROJECT_V1_ROUTE_DEPLOYMENT_CONFIG,
          type: "div",
          hide: !isTenantProject.value,
        },
        {
          title: t("common.change-history"),
          path: PROJECT_V1_ROUTE_CHANGE_HISTORIES,
          type: "div",
          hide: isTenantProject.value,
          permissions: ["bb.changeHistories.list"],
        },
        {
          title: startCase(t("slow-query.slow-queries")),
          path: PROJECT_V1_ROUTE_SLOW_QUERIES,
          type: "div",
          permissions: ["bb.slowQueries.list"],
        },
        {
          title: t("common.anomalies"),
          path: PROJECT_V1_ROUTE_ANOMALIES,
          type: "div",
        },
      ],
    },
    {
      title: t("common.issues"),
      path: PROJECT_V1_ROUTE_ISSUES,
      icon: h(CircleDot),
      type: "div",
      hide: isDefaultProject.value,
      permissions: ["bb.issues.list"],
    },
    {
      title: t("common.branches"),
      path: PROJECT_V1_ROUTE_BRANCHES,
      icon: h(GitBranch),
      type: "div",
      hide: isDefaultProject.value,
      permissions: ["bb.branches.list"],
    },
    {
      title: t("changelist.changelists"),
      path: PROJECT_V1_ROUTE_CHANGELISTS,
      icon: h(PencilRuler),
      type: "div",
      hide: isDefaultProject.value,
      permissions: ["bb.changelists.list"],
    },
    {
      title: t("database.sync-schema.title"),
      path: PROJECT_V1_ROUTE_SYNC_SCHEMA,
      icon: h(RefreshCcw),
      type: "div",
      hide: isDefaultProject.value,
      permissions: ["bb.databases.sync"],
    },
    {
      title: t("settings.sidebar.integration"),
      icon: h(Link),
      type: "div",
      hide: isDefaultProject.value,
      children: [
        {
          title: t("common.gitops"),
          path: PROJECT_V1_ROUTE_GITOPS,
          type: "div",
        },
        {
          title: t("common.webhooks"),
          path: PROJECT_V1_ROUTE_WEBHOOKS,
          type: "div",
        },
      ],
    },
    {
      title: t("common.manage"),
      icon: h(Users),
      type: "div",
      hide: isDefaultProject.value,
      children: [
        {
          title: t("common.members"),
          path: PROJECT_V1_ROUTE_MEMBERS,
          type: "div",
          permissions: ["bb.projects.getIamPolicy"],
        },
        {
          title: t("common.activities"),
          path: PROJECT_V1_ROUTE_ACTIVITIES,
          type: "div",
        },
      ],
    },
    {
      title: t("common.setting"),
      icon: h(Settings),
      path: PROJECT_V1_ROUTE_SETTINGS,
      type: "div",
      hide: isDefaultProject.value,
    },
  ];

  return filterprojectSidebarByPermissions(sidebarList);
});

const getItemClass = (path: string | undefined) => {
  const list = ["outline-item"];
  if (route.name === path) {
    list.push("router-link-active", "bg-link-hover");
    return list;
  }

  switch (route.name) {
    case PROJECT_V1_ROUTE_WEBHOOK_CREATE:
    case PROJECT_V1_ROUTE_WEBHOOK_DETAIL:
      if (path === PROJECT_V1_ROUTE_WEBHOOKS) {
        list.push("router-link-active", "bg-link-hover");
      }
      break;
    case PROJECT_V1_ROUTE_BRANCH_DETAIL:
    case PROJECT_V1_ROUTE_BRANCH_ROLLOUT:
    case PROJECT_V1_ROUTE_BRANCH_MERGE:
    case PROJECT_V1_ROUTE_BRANCH_REBASE:
      if (path === PROJECT_V1_ROUTE_BRANCHES) {
        list.push("router-link-active", "bg-link-hover");
      }
      break;
    case PROJECT_V1_ROUTE_CHANGELIST_DETAIL:
      if (path === PROJECT_V1_ROUTE_CHANGELISTS) {
        list.push("router-link-active", "bg-link-hover");
      }
      break;
    case PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL:
    case PROJECT_V1_ROUTE_DATABASE_GROUP_TABLE_GROUP_DETAIL:
      if (path === PROJECT_V1_ROUTE_DATABASE_GROUPS) {
        list.push("router-link-active", "bg-link-hover");
      }
      break;
    case "workspace.issue.detail":
      if (path === PROJECT_V1_ROUTE_ISSUES) {
        list.push("router-link-active", "bg-link-hover");
      }
      break;
    case "workspace.database.history.detail":
      if (path === PROJECT_V1_ROUTE_CHANGE_HISTORIES) {
        list.push("router-link-active", "bg-link-hover");
      }
      break;
    case "workspace.database.detail":
      if (path === PROJECT_V1_ROUTE_DATABASES) {
        list.push("router-link-active", "bg-link-hover");
      }
      break;
  }
  return list;
};

const onSelect = (path: string | undefined, e: MouseEvent | undefined) => {
  if (!path) {
    return;
  }
  const route = router.resolve({
    name: path,
    params: {
      projectId: getProjectName(project.value.name),
    },
  });

  if (e?.ctrlKey || e?.metaKey) {
    window.open(route.fullPath, "_blank");
  } else {
    router.replace(route);
  }
};

const flattenNavigationItems = computed(() => {
  return projectSidebarItemList.value.flatMap<{
    path: string;
    title: string;
    hide?: boolean;
  }>((item) => {
    if (item.children && item.children.length > 0) {
      return item.children.map((child) => ({
        path: child.path ?? "",
        title: child.title,
        hide: child.hide,
      }));
    }
    return [
      {
        path: item.path ?? "",
        title: item.title,
        hide: item.hide,
      },
    ];
  });
});

const navigationKbarActions = computed(() => {
  const actions = flattenNavigationItems.value.map((item) =>
    defineAction({
      id: `bb.navigation.project.${project.value.uid}.${item.path}`,
      name: item.title,
      section: t("kbar.navigation"),
      keywords: [item.title.toLowerCase(), item.path].join(" "),
      perform: () => onSelect(item.path, undefined),
    })
  );
  return actions;
});
useRegisterActions(navigationKbarActions);

useProjectDatabaseActions(project);
</script>
