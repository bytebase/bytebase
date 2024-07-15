import {
  BuildingIcon,
  GalleryHorizontalEndIcon,
  LayersIcon,
  SquareStackIcon,
} from "lucide-vue-next";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, type RouteRecordRaw } from "vue-router";
import type { SidebarItem } from "@/components/CommonSidebar.vue";
import sqlEditorRoutes, {
  SQL_EDITOR_SETTING_GENERAL_MODULE,
  SQL_EDITOR_SETTING_ENVIRONMENT_MODULE,
  SQL_EDITOR_SETTING_INSTANCE_MODULE,
  SQL_EDITOR_SETTING_PROJECT_MODULE,
} from "@/router/sqlEditor";
import { useCurrentUserV1, usePageMode } from "@/store";
import type { WorkspacePermission } from "@/types";
import { hasWorkspacePermissionV2 } from "@/utils";

export const useSidebarItems = () => {
  const route = useRoute();
  const { t } = useI18n();
  const me = useCurrentUserV1();
  const pageMode = usePageMode();

  const getItemClass = (item: SidebarItem) => {
    if (route.name === item.name) {
      return ["router-link-active", "bg-link-hover"];
    }
    return [];
  };

  const getFlattenRoutes = (
    routes: RouteRecordRaw[],
    permissions: WorkspacePermission[] = []
  ): {
    name: string;
    permissions: WorkspacePermission[];
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
      [] as { name: string; permissions: WorkspacePermission[] }[]
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
          hasWorkspacePermissionV2(me.value, permission)
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
    if (pageMode.value === "STANDALONE") {
      // Hide SQL Editor settings entirely in STANDALONE mode
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
        title: t("common.instances"),
        icon: () => h(LayersIcon),
        name: SQL_EDITOR_SETTING_INSTANCE_MODULE,
        type: "route",
      },
      {
        title: t("common.projects"),
        icon: () => h(GalleryHorizontalEndIcon),
        name: SQL_EDITOR_SETTING_PROJECT_MODULE,
        type: "route",
      },
      {
        title: t("common.environments"),
        icon: () => h(SquareStackIcon),
        name: SQL_EDITOR_SETTING_ENVIRONMENT_MODULE,
        type: "route",
      },
    ];

    return filterSidebarByPermissions(sidebarList);
  });

  return { itemList };
};
