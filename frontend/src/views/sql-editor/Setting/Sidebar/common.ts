import {
  BuildingIcon,
  GalleryHorizontalEndIcon,
  LayersIcon,
  ShieldCheckIcon,
  SquareStackIcon,
  UserCircleIcon,
  UsersIcon,
} from "lucide-vue-next";
import { computed, h, unref, type MaybeRef } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { DatabaseIcon } from "@/components/Icon";
import type { SidebarItem } from "@/components/v2/Sidebar/CommonSidebar.vue";
import { getFlattenRoutes } from "@/components/v2/Sidebar/utils";
import sqlEditorRoutes, {
  SQL_EDITOR_SETTING_AUDIT_LOG_MODULE,
  SQL_EDITOR_SETTING_DATA_CLASSIFICATION_MODULE,
  SQL_EDITOR_SETTING_GLOBAL_MASKING_MODULE,
  SQL_EDITOR_SETTING_DATA_SEMANTIC_TYPES,
  SQL_EDITOR_SETTING_DATABASES_MODULE,
  SQL_EDITOR_SETTING_ENVIRONMENT_MODULE,
  SQL_EDITOR_SETTING_GENERAL_MODULE,
  SQL_EDITOR_SETTING_INSTANCE_MODULE,
  SQL_EDITOR_SETTING_MEMBERS_MODULE,
  SQL_EDITOR_SETTING_PROFILE_MODULE,
  SQL_EDITOR_SETTING_PROJECT_MODULE,
  SQL_EDITOR_SETTING_ROLES_MODULE,
  SQL_EDITOR_SETTING_SSO_MODULE,
  SQL_EDITOR_SETTING_SUBSCRIPTION_MODULE,
  SQL_EDITOR_SETTING_USERS_MODULE,
} from "@/router/sqlEditor";
import { useAppFeature, usePermissionStore } from "@/store";
import { hasWorkspacePermissionV2 } from "@/utils";

export const useSidebarItems = (ignoreModeCheck?: MaybeRef<boolean>) => {
  const route = useRoute();
  const permissionStore = usePermissionStore();
  const { t } = useI18n();
  const disableSetting = useAppFeature("bb.feature.sql-editor.disable-setting");

  const getItemClass = (item: SidebarItem) => {
    if (route.name === item.name) {
      return ["router-link-active", "bg-link-hover"];
    }
    return [];
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
    if (disableSetting.value && !unref(ignoreModeCheck)) {
      // Hide SQL Editor settings entirely if
      // - embedded in iframe
      // - or workspace mode is Issue mode
      return [];
    }

    const sidebarList: SidebarItem[] = [
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
      },
      {
        title: t("settings.sidebar.workspace"),
        icon: () => h(BuildingIcon),
        type: "div",
        expand: true,
        children: [
          {
            title: t("settings.sidebar.general"),
            name: SQL_EDITOR_SETTING_GENERAL_MODULE,
            type: "route",
          },
          {
            title: t("settings.sidebar.subscription"),
            name: SQL_EDITOR_SETTING_SUBSCRIPTION_MODULE,
            type: "route",
          },
        ],
      },
      {
        title: t("settings.sidebar.account"),
        icon: () => h(UserCircleIcon),
        type: "div",
        expand: true,
        children: [
          {
            title: t("settings.sidebar.profile"),
            name: SQL_EDITOR_SETTING_PROFILE_MODULE,
            type: "route",
          },
        ],
      },
      {
        type: "divider",
      },
      {
        title: t("settings.sidebar.iam-and-admin"),
        icon: () => h(UsersIcon),
        type: "div",
        expand: true,
        children: [
          {
            title: t("settings.sidebar.users-and-groups"),
            name: SQL_EDITOR_SETTING_USERS_MODULE,
            type: "route",
          },
          {
            title: t("settings.sidebar.members"),
            name: SQL_EDITOR_SETTING_MEMBERS_MODULE,
            type: "route",
            hide: permissionStore.onlyWorkspaceMember,
          },
          {
            title: t("settings.sidebar.custom-roles"),
            name: SQL_EDITOR_SETTING_ROLES_MODULE,
            type: "route",
          },
          {
            title: t("settings.sidebar.sso"),
            name: SQL_EDITOR_SETTING_SSO_MODULE,
            type: "route",
          },
          {
            title: t("settings.sidebar.audit-log"),
            name: SQL_EDITOR_SETTING_AUDIT_LOG_MODULE,
            type: "route",
          },
        ],
      },
      {
        title: t("settings.sidebar.data-access"),
        icon: () => h(ShieldCheckIcon),
        type: "div",
        expand: true,
        children: [
          {
            title: t("settings.sensitive-data.semantic-types.self"),
            name: SQL_EDITOR_SETTING_DATA_SEMANTIC_TYPES,
            type: "route",
          },
          {
            title: t("settings.sidebar.data-classification"),
            name: SQL_EDITOR_SETTING_DATA_CLASSIFICATION_MODULE,
            type: "route",
          },
          {
            title: t("settings.sidebar.global-masking"),
            name: SQL_EDITOR_SETTING_GLOBAL_MASKING_MODULE,
            type: "route",
          },
        ],
      },
    ];

    return filterSidebarByPermissions(sidebarList);
  });

  return { itemList };
};
