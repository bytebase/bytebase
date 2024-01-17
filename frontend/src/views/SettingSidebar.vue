<template>
  <CommonSidebar
    :key="'setting'"
    :item-list="(settingSidebarItemList as SidebarItem[])"
    :get-item-class="getItemClass"
    @select="onSelect"
  />
</template>

<script lang="ts" setup>
import {
  UserCircle,
  Building,
  ShieldCheck,
  Link,
  Archive,
} from "lucide-vue-next";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { RouteRecordRaw, useRoute, useRouter } from "vue-router";
import { SidebarItem } from "@/components/CommonSidebar.vue";
import workspaceSettingRoutes, {
  SETTING_ROUTE_PROFILE,
  SETTING_ROUTE_WORKSPACE_GENERAL,
  SETTING_ROUTE_WORKSPACE_MEMBER,
  SETTING_ROUTE_WORKSPACE_ROLE,
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
  SETTING_ROUTE_WORKSPACE_DEBUG_LOG,
  SETTING_ROUTE_WORKSPACE_SQL_REVIEW,
  SETTING_ROUTE_WORKSPACE_SQL_REVIEW_CREATE,
  SETTING_ROUTE_WORKSPACE_SQL_REVIEW_DETAIL,
  SETTING_ROUTE_WORKSPACE_SLOW_QUERY,
  SETTING_ROUTE_WORKSPACE_SCHEMA_TEMPLATE,
  SETTING_ROUTE_WORKSPACE_RISK_CENTER,
  SETTING_ROUTE_WORKSPACE_CUSTOM_APPROVAL,
  SETTING_ROUTE_WORKSPACE_SENSITIVE_DATA,
  SETTING_ROUTE_WORKSPACE_ACCESS_CONTROL,
  SETTING_ROUTE_WORKSPACE_AUDIT_LOG,
  SETTING_ROUTE_WORKSPACE_GITOPS,
  SETTING_ROUTE_WORKSPACE_GITOPS_CREATE,
  SETTING_ROUTE_WORKSPACE_GITOPS_DETAIL,
  SETTING_ROUTE_WORKSPACE_SSO,
  SETTING_ROUTE_WORKSPACE_SSO_CREATE,
  SETTING_ROUTE_WORKSPACE_SSO_DETAIL,
  SETTING_ROUTE_WORKSPACE_MAIL_DELIVERY,
  SETTING_ROUTE_WORKSPACE_ARCHIVE,
  SETTING_ROUTE_WORKSPACE,
} from "@/router/dashboard/workspaceSetting";
import { useCurrentUserV1 } from "@/store";
import {
  hasWorkspaceLevelProjectPermission,
  hasWorkspacePermissionV2,
} from "@/utils";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const currentUser = useCurrentUserV1();

const getItemClass = (item: SidebarItem) => {
  const list = [];
  if (route.name === item.name) {
    list.push("router-link-active", "bg-link-hover");
    return list;
  }

  switch (route.name) {
    case SETTING_ROUTE_WORKSPACE_SSO_CREATE:
    case SETTING_ROUTE_WORKSPACE_SSO_DETAIL:
      if (item.name === SETTING_ROUTE_WORKSPACE_SSO) {
        list.push("router-link-active", "bg-link-hover");
      }
      break;
    case SETTING_ROUTE_WORKSPACE_SQL_REVIEW_CREATE:
    case SETTING_ROUTE_WORKSPACE_SQL_REVIEW_DETAIL:
      if (item.name === SETTING_ROUTE_WORKSPACE_SQL_REVIEW) {
        list.push("router-link-active", "bg-link-hover");
      }
      break;
    case SETTING_ROUTE_WORKSPACE_GITOPS_CREATE:
    case SETTING_ROUTE_WORKSPACE_GITOPS_DETAIL:
      if (item.name === SETTING_ROUTE_WORKSPACE_GITOPS) {
        list.push("router-link-active", "bg-link-hover");
      }
      break;
  }
  return list;
};

const onSelect = (item: SidebarItem, e: MouseEvent | undefined) => {
  if (!item.name) {
    return;
  }
  const route = router.resolve({
    name: item.name,
  });

  if (e?.ctrlKey || e?.metaKey) {
    window.open(route.fullPath, "_blank");
  } else {
    router.replace(route);
  }
};

const settingSidebarItemList = computed((): SidebarItem[] => {
  const list: SidebarItem[] = [
    {
      title: t("settings.sidebar.account"),
      icon: h(UserCircle),
      type: "div",
      children: [
        {
          title: t("settings.sidebar.profile"),
          name: SETTING_ROUTE_PROFILE,
          type: "route",
        },
      ],
    },
    {
      title: t("settings.sidebar.workspace"),
      icon: h(Building),
      type: "div",
      children: [
        {
          title: t("settings.sidebar.general"),
          name: SETTING_ROUTE_WORKSPACE_GENERAL,
          type: "route",
        },
        {
          title: t("settings.sidebar.members"),
          name: SETTING_ROUTE_WORKSPACE_MEMBER,
          type: "route",
        },
        {
          title: t("settings.sidebar.custom-roles"),
          name: SETTING_ROUTE_WORKSPACE_ROLE,
          type: "route",
        },
        {
          title: t("settings.sidebar.subscription"),
          name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
          type: "route",
        },
        {
          title: t("settings.sidebar.debug-log"),
          name: SETTING_ROUTE_WORKSPACE_DEBUG_LOG,
          type: "route",
        },
      ],
    },
    {
      title: t("settings.sidebar.security-and-policy"),
      icon: h(ShieldCheck),
      type: "div",
      children: [
        {
          title: t("sql-review.title"),
          name: SETTING_ROUTE_WORKSPACE_SQL_REVIEW,
          type: "route",
        },
        {
          title: t("slow-query.self"),
          name: SETTING_ROUTE_WORKSPACE_SLOW_QUERY,
          type: "route",
        },
        {
          title: t("schema-template.self"),
          name: SETTING_ROUTE_WORKSPACE_SCHEMA_TEMPLATE,
          type: "route",
        },
        {
          title: t("custom-approval.risk.risk-center"),
          name: SETTING_ROUTE_WORKSPACE_RISK_CENTER,
          type: "route",
        },
        {
          title: t("custom-approval.self"),
          name: SETTING_ROUTE_WORKSPACE_CUSTOM_APPROVAL,
          type: "route",
        },
        {
          title: t("settings.sidebar.sensitive-data"),
          name: SETTING_ROUTE_WORKSPACE_SENSITIVE_DATA,
          type: "route",
        },
        {
          title: t("settings.sidebar.access-control"),
          name: SETTING_ROUTE_WORKSPACE_ACCESS_CONTROL,
          type: "route",
        },
        {
          title: t("settings.sidebar.audit-log"),
          name: SETTING_ROUTE_WORKSPACE_AUDIT_LOG,
          type: "route",
        },
      ],
    },
    {
      title: t("settings.sidebar.integration"),
      icon: h(Link),
      type: "div",
      children: [
        {
          title: t("settings.sidebar.gitops"),
          name: SETTING_ROUTE_WORKSPACE_GITOPS,
          type: "route",
        },
        {
          title: t("settings.sidebar.sso"),
          name: SETTING_ROUTE_WORKSPACE_SSO,
          type: "route",
        },
        {
          title: t("settings.sidebar.mail-delivery"),
          name: SETTING_ROUTE_WORKSPACE_MAIL_DELIVERY,
          type: "route",
        },
      ],
    },
    {
      title: t("common.archived"),
      icon: h(Archive),
      name: SETTING_ROUTE_WORKSPACE_ARCHIVE,
      type: "route",
    },
  ];

  const checkSidebarItemPermission = (item: SidebarItem) => {
    if (item.name) {
      item.hide = !hasRoutePermission(item.name);
    }
    if (item.children) {
      item.children.forEach((child) => {
        checkSidebarItemPermission(child);
      });
    }
  };
  list.forEach((item) => {
    checkSidebarItemPermission(item);
  });

  return list;
});

const routes = workspaceSettingRoutes.find(
  (route) => route.name === SETTING_ROUTE_WORKSPACE
)?.children as RouteRecordRaw[];

const hasRoutePermission = (routeName: string) => {
  const route = routes.find((route) => route.name === routeName);
  if (!route) {
    return false;
  }

  if (route.meta?.requiredWorkspacePermissionList) {
    const requiredPermissions = route.meta.requiredWorkspacePermissionList();
    return requiredPermissions.every((permission) =>
      hasWorkspacePermissionV2(currentUser.value, permission)
    );
  } else if (route.meta?.requiredProjectPermissionList) {
    const requiredPermissions = route.meta.requiredProjectPermissionList();
    return requiredPermissions.every((permission) =>
      hasWorkspaceLevelProjectPermission(currentUser.value, permission)
    );
  }

  return true;
};
</script>
