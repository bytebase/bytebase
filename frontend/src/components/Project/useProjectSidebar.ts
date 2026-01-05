import {
  CircleDot,
  Database,
  DownloadIcon,
  LayoutList,
  PackageIcon,
  PlayCircle,
  Settings,
  Users,
  Workflow,
} from "lucide-vue-next";
import { computed, h, type MaybeRef, unref } from "vue";
import { useRoute } from "vue-router";
import type { SidebarItem } from "@/components/v2/Sidebar/type";
import { useIssueLayoutVersion } from "@/composables/useIssueLayoutVersion";
import { t } from "@/plugins/i18n";
import {
  PROJECT_V1_ROUTE_AUDIT_LOGS,
  PROJECT_V1_ROUTE_DATABASE_GROUPS,
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_EXPORT_CENTER,
  PROJECT_V1_ROUTE_ISSUES,
  PROJECT_V1_ROUTE_MASKING_EXEMPTION,
  PROJECT_V1_ROUTE_MEMBERS,
  PROJECT_V1_ROUTE_PLANS,
  PROJECT_V1_ROUTE_RELEASES,
  PROJECT_V1_ROUTE_ROLLOUTS,
  PROJECT_V1_ROUTE_SETTINGS,
  PROJECT_V1_ROUTE_SYNC_SCHEMA,
  PROJECT_V1_ROUTE_WEBHOOKS,
} from "@/router/dashboard/projectV1";
import { DEFAULT_PROJECT_NAME } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

interface ProjectSidebarItem extends SidebarItem {
  title: string;
  type: "div";
  hide?: boolean;
  children?: ProjectSidebarItem[];
}

export const useProjectSidebar = (project: MaybeRef<Project>) => {
  const route = useRoute();
  const { enabledNewLayout } = useIssueLayoutVersion();

  const isDefaultProject = computed((): boolean => {
    return unref(project).name === DEFAULT_PROJECT_NAME;
  });

  const projectSidebarItemList = computed((): ProjectSidebarItem[] => {
    const cicdRoutes: ProjectSidebarItem[] = enabledNewLayout.value
      ? [
          {
            title: "CI/CD",
            icon: () => h(Workflow),
            type: "div",
            expand: true,
            hide: isDefaultProject.value,
            children: [
              {
                title: t("plan.plans"),
                path: PROJECT_V1_ROUTE_PLANS,
                type: "div",
              },
              {
                title: t("rollout.rollouts"),
                path: PROJECT_V1_ROUTE_ROLLOUTS,
                type: "div",
              },
              {
                title: t("release.releases"),
                path: PROJECT_V1_ROUTE_RELEASES,
                type: "div",
              },
            ],
          },
        ]
      : [
          {
            title: t("release.releases"),
            path: PROJECT_V1_ROUTE_RELEASES,
            icon: () => h(PackageIcon),
            type: "div",
            hide: isDefaultProject.value,
          },
          {
            title: t("plan.plans"),
            icon: () => h(LayoutList),
            path: PROJECT_V1_ROUTE_PLANS,
            type: "div",
            hide: isDefaultProject.value,
          },
          {
            title: t("rollout.rollouts"),
            path: PROJECT_V1_ROUTE_ROLLOUTS,
            icon: () => h(PlayCircle),
            type: "div",
            hide: isDefaultProject.value,
          },
        ];

    const databaseRoutes: ProjectSidebarItem[] = [
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
            title: t("database.sync-schema.title"),
            path: PROJECT_V1_ROUTE_SYNC_SCHEMA,
            type: "div",
          },
        ],
      },
    ];

    const sidebarList: ProjectSidebarItem[] = [
      {
        title: t("common.issues"),
        path: PROJECT_V1_ROUTE_ISSUES,
        icon: () => h(CircleDot),
        type: "div",
        hide: isDefaultProject.value,
      },
      ...(enabledNewLayout.value
        ? [...cicdRoutes, ...databaseRoutes]
        : [...databaseRoutes, ...cicdRoutes]),
      {
        title: t("export-center.self"),
        icon: () => h(DownloadIcon),
        path: PROJECT_V1_ROUTE_EXPORT_CENTER,
        type: "div",
        hide: isDefaultProject.value,
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
            title: t("common.webhooks"),
            path: PROJECT_V1_ROUTE_WEBHOOKS,
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
        title: t("common.setting"),
        icon: () => h(Settings),
        path: PROJECT_V1_ROUTE_SETTINGS,
        type: "div",
        hide: isDefaultProject.value,
      },
    ];

    return sidebarList;
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
    const currentRoute = current?.toString() ?? "";
    const isActiveRoute =
      item.path === currentRoute || currentRoute.startsWith(`${item.path}.`);

    return isActiveRoute;
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
