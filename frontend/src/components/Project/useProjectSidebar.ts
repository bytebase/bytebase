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
  SearchCodeIcon,
  DownloadIcon,
} from "lucide-vue-next";
import { computed, h } from "vue";
import type { Ref } from "vue";
import type { RouteRecordRaw } from "vue-router";
import { useRoute } from "vue-router";
import type { SidebarItem } from "@/components/CommonSidebar.vue";
import { t } from "@/plugins/i18n";
import projectV1Routes, {
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_ISSUES,
  PROJECT_V1_ROUTE_CHANGE_HISTORIES,
  PROJECT_V1_ROUTE_DATABASE_CHANGE_HISTORY_DETAIL,
  PROJECT_V1_ROUTE_SYNC_SCHEMA,
  PROJECT_V1_ROUTE_SLOW_QUERIES,
  PROJECT_V1_ROUTE_ANOMALIES,
  PROJECT_V1_ROUTE_GITOPS,
  PROJECT_V1_ROUTE_MEMBERS,
  PROJECT_V1_ROUTE_SETTINGS,
  PROJECT_V1_ROUTE_WEBHOOKS,
  PROJECT_V1_ROUTE_BRANCHES,
  PROJECT_V1_ROUTE_CHANGELISTS,
  PROJECT_V1_ROUTE_DATABASE_GROUPS,
  PROJECT_V1_ROUTE_DEPLOYMENT_CONFIG,
  PROJECT_V1_ROUTE_EXPORT_CENTER,
  PROJECT_V1_ROUTE_AUDIT_LOGS,
  PROJECT_V1_ROUTE_REVIEW_CENTER,
} from "@/router/dashboard/projectV1";
import { useCurrentUserV1 } from "@/store";
import type { ComposedProject } from "@/types";
import { DEFAULT_PROJECT_V1_NAME } from "@/types";
import type { ProjectPermission } from "@/types";
import { TenantMode } from "@/types/proto/v1/project_service";
import { hasProjectPermissionV2 } from "@/utils";

interface ProjectSidebarItem extends SidebarItem {
  title: string;
  type: "div";
  hide?: boolean;
  children?: ProjectSidebarItem[];
}

export const useProjectSidebar = (project: Ref<ComposedProject>) => {
  const currentUser = useCurrentUserV1();
  const route = useRoute();

  const isDefaultProject = computed((): boolean => {
    return project.value.name === DEFAULT_PROJECT_V1_NAME;
  });

  const isTenantProject = computed((): boolean => {
    return project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED;
  });

  const getFlattenProjectV1Routes = (
    routes: RouteRecordRaw[],
    permissions: ProjectPermission[] = []
  ): {
    name: string;
    permissions: ProjectPermission[];
  }[] => {
    return routes.reduce(
      (list, projectV1Route) => {
        const requiredProjectPermissionListFunc =
          projectV1Route.meta?.requiredProjectPermissionList;
        let requiredPermissionList = requiredProjectPermissionListFunc
          ? requiredProjectPermissionListFunc()
          : [];
        if (requiredPermissionList.length === 0) {
          requiredPermissionList = permissions;
        }

        if (projectV1Route.name && projectV1Route.name.toString() !== "") {
          list.push({
            name: projectV1Route.name.toString(),
            permissions: requiredPermissionList,
          });
        }
        if (projectV1Route.children) {
          list.push(
            ...getFlattenProjectV1Routes(
              projectV1Route.children,
              requiredPermissionList
            )
          );
        }
        return list;
      },
      [] as { name: string; permissions: ProjectPermission[] }[]
    );
  };

  const flattenProjectV1Routes = computed(() => {
    return getFlattenProjectV1Routes(projectV1Routes);
  });

  const filterProjectSidebarByPermissions = (
    sidebarList: ProjectSidebarItem[]
  ): ProjectSidebarItem[] => {
    return sidebarList
      .filter((item) => {
        const routeConfig = flattenProjectV1Routes.value.find(
          (projectV1Route) => projectV1Route.name === item.path
        );
        return (routeConfig?.permissions ?? []).every((permission) =>
          hasProjectPermissionV2(project.value, currentUser.value, permission)
        );
      })
      .map((item) => ({
        ...item,
        children: filterProjectSidebarByPermissions(item.children ?? []),
      }));
  };

  const projectSidebarItemList = computed((): ProjectSidebarItem[] => {
    const sidebarList: ProjectSidebarItem[] = [
      {
        title: t("common.database"),
        icon: () => h(Database),
        type: "div",
        expand: true,
        children: [
          {
            title: t("common.databases"),
            path: PROJECT_V1_ROUTE_DATABASES,
            type: "div",
          },
          {
            title: t("common.groups"),
            path: PROJECT_V1_ROUTE_DATABASE_GROUPS,
            type: "div",
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
          },
          {
            title: startCase(t("slow-query.slow-queries")),
            path: PROJECT_V1_ROUTE_SLOW_QUERIES,
            type: "div",
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
        icon: () => h(CircleDot),
        type: "div",
        hide: isDefaultProject.value,
      },
      {
        title: t("review-center.self"),
        icon: () => h(SearchCodeIcon),
        path: PROJECT_V1_ROUTE_REVIEW_CENTER,
        type: "div",
        hide: isDefaultProject.value,
      },
      {
        title: t("export-center.self"),
        icon: () => h(DownloadIcon),
        path: PROJECT_V1_ROUTE_EXPORT_CENTER,
        type: "div",
        hide: isDefaultProject.value,
      },
      {
        title: t("common.branches"),
        path: PROJECT_V1_ROUTE_BRANCHES,
        icon: () => h(GitBranch),
        type: "div",
        hide: isDefaultProject.value,
      },
      {
        title: t("changelist.changelists"),
        path: PROJECT_V1_ROUTE_CHANGELISTS,
        icon: () => h(PencilRuler),
        type: "div",
        hide: isDefaultProject.value,
      },
      {
        title: t("database.sync-schema.title"),
        path: PROJECT_V1_ROUTE_SYNC_SCHEMA,
        icon: () => h(RefreshCcw),
        type: "div",
        hide: isDefaultProject.value,
      },
      {
        title: t("settings.sidebar.integration"),
        icon: () => h(Link),
        type: "div",
        hide: isDefaultProject.value,
        expand: true,
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
        icon: () => h(Users),
        type: "div",
        hide: isDefaultProject.value,
        expand: true,
        children: [
          {
            title: t("common.members"),
            path: PROJECT_V1_ROUTE_MEMBERS,
            type: "div",
          },
          {
            title: t("settings.sidebar.audit-log"),
            path: PROJECT_V1_ROUTE_AUDIT_LOGS,
            type: "div",
          },
        ],
      },
      {
        title: t("common.setting"),
        icon: () => h(Settings),
        path: PROJECT_V1_ROUTE_SETTINGS,
        type: "div",
        hide: isDefaultProject.value,
      },
    ];

    return filterProjectSidebarByPermissions(sidebarList);
  });

  const flattenNavigationItems = computed(() => {
    return projectSidebarItemList.value.flatMap<ProjectSidebarItem>((item) => {
      if (item.children && item.children.length > 0) {
        return item.children.map((child) => ({
          ...child,
          hide: item.hide || child.hide,
        }));
      }
      return item;
    });
  });

  const checkIsActive = (item: SidebarItem) => {
    const { name: current } = route;

    if (
      current?.toString() === PROJECT_V1_ROUTE_DATABASE_CHANGE_HISTORY_DETAIL
    ) {
      if (item.path === PROJECT_V1_ROUTE_CHANGE_HISTORIES) {
        return true;
      }
      return false;
    }

    const isActiveRoute =
      item.path === current?.toString() ||
      current?.toString().startsWith(`${item.path}.`);

    if (isActiveRoute) {
      return true;
    }
    return false;
  };

  const activeSidebar = computed(() => {
    return flattenNavigationItems.value
      .filter((item) => !item.hide && item.path)
      .find((item) => checkIsActive(item));
  });

  return {
    projectSidebarItemList,
    flattenNavigationItems,
    activeSidebar,
    checkIsActive,
  };
};
