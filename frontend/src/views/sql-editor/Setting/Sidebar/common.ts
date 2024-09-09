import {
  BuildingIcon,
  GalleryHorizontalEndIcon,
  LayersIcon,
  ShieldCheckIcon,
  SquareStackIcon,
  UsersIcon,
  WorkflowIcon,
} from "lucide-vue-next";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter, type RouteRecordRaw } from "vue-router";
import type { SidebarItem } from "@/components/CommonSidebar.vue";
import { DatabaseIcon } from "@/components/Icon";
import {
  WORKSPACE_ROUTE_DATA_CLASSIFICATION,
  WORKSPACE_ROUTE_SQL_REVIEW,
  WORKSPACE_ROUTE_USERS,
} from "@/router/dashboard/workspaceRoutes";
import sqlEditorRoutes, {
  SQL_EDITOR_SETTING_DATABASES_MODULE,
  SQL_EDITOR_SETTING_ENVIRONMENT_MODULE,
  SQL_EDITOR_SETTING_GENERAL_MODULE,
  SQL_EDITOR_SETTING_INSTANCE_MODULE,
  SQL_EDITOR_SETTING_PROJECT_MODULE,
} from "@/router/sqlEditor";
import { useAppFeature } from "@/store";
import type { Permission } from "@/types";
import { hasWorkspacePermissionV2 } from "@/utils";

export const useSidebarItems = () => {
  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const disableSetting = useAppFeature("bb.feature.sql-editor.disable-setting");

  const getItemClass = (item: SidebarItem) => {
    if (route.name === item.name) {
      return ["router-link-active", "bg-link-hover"];
    }
    return [];
  };

  const getFlattenRoutes = (
    routes: RouteRecordRaw[],
    permissions: Permission[] = []
  ): {
    name: string;
    permissions: Permission[];
  }[] => {
    return routes.reduce(
      (list, workspaceRoute) => {
        const requiredWorkspacePermissionListFunc =
          workspaceRoute.meta?.requiredWorkspacePermissionList;
        let requiredPermissionList = requiredWorkspacePermissionListFunc
          ? requiredWorkspacePermissionListFunc()
          : [];
        if (requiredPermissionList.length === 0) {
          requiredPermissionList = permissions;
        }

        if (workspaceRoute.name && workspaceRoute.name.toString() !== "") {
          list.push({
            name: workspaceRoute.name.toString(),
            permissions: requiredPermissionList,
          });
        }
        if (workspaceRoute.children) {
          list.push(
            ...getFlattenRoutes(workspaceRoute.children, requiredPermissionList)
          );
        }
        return list;
      },
      [] as { name: string; permissions: Permission[] }[]
    );
  };

  const flattenRoutes = computed(() => {
    return getFlattenRoutes(sqlEditorRoutes);
  });

  const filterSidebarByPermissions = (items: SidebarItem[]): SidebarItem[] => {
    return items
      .filter((item) => {
        const routeConfig = flattenRoutes.value.find(
          (workspaceRoute) => workspaceRoute.name === item.name
        );
        return (routeConfig?.permissions ?? []).every((permission) =>
          hasWorkspacePermissionV2(permission)
        );
      })
      .map((item) => ({
        ...item,
        expand:
          item.expand ||
          (item.children ?? [])
            .reduce((classList, child) => {
              classList.push(...getItemClass(child));
              return classList;
            }, [] as string[])
            .includes("router-link-active"),
        children: filterSidebarByPermissions(item.children ?? []),
      }));
  };

  const itemList = computed((): SidebarItem[] => {
    if (disableSetting.value) {
      // Hide SQL Editor settings entirely if embedded in iframe
      return [];
    }

    const sidebarList: SidebarItem[] = [
      {
        title: t("settings.sidebar.general"),
        icon: () => h(BuildingIcon),
        name: SQL_EDITOR_SETTING_GENERAL_MODULE,
        type: "route",
      },
      {
        title: t("common.projects"),
        icon: () => h(GalleryHorizontalEndIcon),
        name: SQL_EDITOR_SETTING_PROJECT_MODULE,
        type: "route",
      },
      {
        title: t("common.instances"),
        icon: () => h(LayersIcon),
        name: SQL_EDITOR_SETTING_INSTANCE_MODULE,
        type: "route",
      },
      {
        title: t("common.databases"),
        icon: () => h(DatabaseIcon),
        name: SQL_EDITOR_SETTING_DATABASES_MODULE,
        type: "route",
      },
      {
        title: t("common.environments"),
        icon: () => h(SquareStackIcon),
        name: SQL_EDITOR_SETTING_ENVIRONMENT_MODULE,
        type: "route",
      },
      {
        type: "divider",
        name: "",
      },
      {
        title: t("settings.sidebar.users-and-groups"),
        icon: () => h(UsersIcon),
        name: WORKSPACE_ROUTE_USERS,
        path: router.resolve({ name: WORKSPACE_ROUTE_USERS }).fullPath,
        type: "link",
        newWindow: true,
      },
      {
        title: "CI/CD",
        icon: () => h(WorkflowIcon),
        name: WORKSPACE_ROUTE_SQL_REVIEW,
        path: router.resolve({ name: WORKSPACE_ROUTE_SQL_REVIEW }).fullPath,
        type: "link",
        newWindow: true,
      },
      {
        title: t("settings.sidebar.data-access"),
        icon: () => h(ShieldCheckIcon),
        name: WORKSPACE_ROUTE_DATA_CLASSIFICATION,
        path: router.resolve({ name: WORKSPACE_ROUTE_DATA_CLASSIFICATION })
          .fullPath,
        type: "link",
        newWindow: true,
      },
    ];

    return filterSidebarByPermissions(sidebarList);
  });

  return { itemList };
};
