import { startCase } from "lodash-es";
import {
  Database,
  CircleDot,
  Users,
  Link,
  Settings,
  RefreshCcw,
  PencilRuler,
  SearchCodeIcon,
  DownloadIcon,
  SquareGanttChartIcon,
  PackageIcon,
} from "lucide-vue-next";
import { computed, h, unref } from "vue";
import type { RouteLocationNormalizedLoaded, RouteRecordRaw } from "vue-router";
import { useRoute } from "vue-router";
import type { SidebarItem } from "@/components/CommonSidebar.vue";
import { t } from "@/plugins/i18n";
import projectV1Routes, {
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_ISSUES,
  PROJECT_V1_ROUTE_SYNC_SCHEMA,
  PROJECT_V1_ROUTE_SLOW_QUERIES,
  PROJECT_V1_ROUTE_ANOMALIES,
  PROJECT_V1_ROUTE_MEMBERS,
  PROJECT_V1_ROUTE_SETTINGS,
  PROJECT_V1_ROUTE_WEBHOOKS,
  PROJECT_V1_ROUTE_CHANGELISTS,
  PROJECT_V1_ROUTE_DATABASE_GROUPS,
  PROJECT_V1_ROUTE_DEPLOYMENT_CONFIG,
  PROJECT_V1_ROUTE_EXPORT_CENTER,
  PROJECT_V1_ROUTE_AUDIT_LOGS,
  PROJECT_V1_ROUTE_REVIEW_CENTER,
  PROJECT_V1_ROUTE_RELEASES,
  PROJECT_V1_ROUTE_MASKING_EXEMPTION,
} from "@/router/dashboard/projectV1";
import { useAppFeature } from "@/store";
import type { ComposedProject, MaybeRef, Permission } from "@/types";
import { DEFAULT_PROJECT_NAME } from "@/types";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";
import { hasProjectPermissionV2 } from "@/utils";

interface ProjectSidebarItem extends SidebarItem {
  title: string;
  type: "div";
  hide?: boolean;
  children?: ProjectSidebarItem[];
}

export const useProjectSidebar = (
  project: MaybeRef<ComposedProject>,
  _route?: RouteLocationNormalizedLoaded
) => {
  const route = _route ?? useRoute();
  const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");

  const isDefaultProject = computed((): boolean => {
    return unref(project).name === DEFAULT_PROJECT_NAME;
  });

  const getFlattenProjectV1Routes = (
    routes: RouteRecordRaw[],
    permissions: Permission[] = []
  ): {
    name: string;
    permissions: Permission[];
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
      [] as { name: string; permissions: Permission[] }[]
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
          hasProjectPermissionV2(unref(project), permission)
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
        title: t("common.issues"),
        path: PROJECT_V1_ROUTE_ISSUES,
        icon: () => h(CircleDot),
        type: "div",
        hide: isDefaultProject.value,
      },
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
            hide: databaseChangeMode.value === DatabaseChangeMode.EDITOR,
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
        title: t("review-center.self"),
        icon: () => h(SearchCodeIcon),
        path: PROJECT_V1_ROUTE_REVIEW_CENTER,
        type: "div",
        hide:
          isDefaultProject.value ||
          databaseChangeMode.value === DatabaseChangeMode.EDITOR,
      },
      {
        title: t("export-center.self"),
        icon: () => h(DownloadIcon),
        path: PROJECT_V1_ROUTE_EXPORT_CENTER,
        type: "div",
        hide:
          isDefaultProject.value ||
          databaseChangeMode.value === DatabaseChangeMode.EDITOR,
      },
      {
        title: t("changelist.changelists"),
        path: PROJECT_V1_ROUTE_CHANGELISTS,
        icon: () => h(PencilRuler),
        type: "div",
        hide:
          isDefaultProject.value ||
          databaseChangeMode.value === DatabaseChangeMode.EDITOR,
      },
      {
        title: t("release.releases"),
        path: PROJECT_V1_ROUTE_RELEASES,
        icon: () => h(PackageIcon),
        type: "div",
        hide:
          isDefaultProject.value ||
          databaseChangeMode.value === DatabaseChangeMode.EDITOR,
      },
      {
        title: t("database.sync-schema.title"),
        path: PROJECT_V1_ROUTE_SYNC_SCHEMA,
        icon: () => h(RefreshCcw),
        type: "div",
        hide:
          isDefaultProject.value ||
          databaseChangeMode.value === DatabaseChangeMode.EDITOR,
      },
      {
        title: t("settings.sidebar.integration"),
        icon: () => h(Link),
        type: "div",
        hide:
          isDefaultProject.value ||
          databaseChangeMode.value === DatabaseChangeMode.EDITOR,
        expand: true,
        children: [
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
            title: t("project.masking-exemption.self"),
            path: PROJECT_V1_ROUTE_MASKING_EXEMPTION,
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
        title: t("common.deployment-config"),
        icon: () => h(SquareGanttChartIcon),
        path: PROJECT_V1_ROUTE_DEPLOYMENT_CONFIG,
        type: "div",
        hide:
          isDefaultProject.value ||
          databaseChangeMode.value === DatabaseChangeMode.EDITOR,
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
