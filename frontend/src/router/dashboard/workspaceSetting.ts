import { RouteRecordRaw } from "vue-router";
import { t } from "@/plugins/i18n";
import SettingSidebar from "@/views/SettingSidebar.vue";

export const SETTING_ROUTE = "setting";
export const SETTING_ROUTE_WORKSPACE = `${SETTING_ROUTE}.workspace`;
export const SETTING_ROUTE_PROFILE = `${SETTING_ROUTE}.profile`;
export const SETTING_ROUTE_PROFILE_TWO_FACTOR = `${SETTING_ROUTE_PROFILE}.two-factor`;
export const SETTING_ROUTE_WORKSPACE_GENERAL = `${SETTING_ROUTE_WORKSPACE}.general`;
export const SETTING_ROUTE_WORKSPACE_MEMBER = `${SETTING_ROUTE_WORKSPACE}.member`;
export const SETTING_ROUTE_WORKSPACE_ROLE = `${SETTING_ROUTE_WORKSPACE}.role`;
export const SETTING_ROUTE_WORKSPACE_SUBSCRIPTION = `${SETTING_ROUTE_WORKSPACE}.subscription`;
export const SETTING_ROUTE_WORKSPACE_DEBUG_LOG = `${SETTING_ROUTE_WORKSPACE}.debug-log`;
export const SETTING_ROUTE_WORKSPACE_ARCHIVE = `${SETTING_ROUTE_WORKSPACE}.archive`;

const workspaceSettingRoutes: RouteRecordRaw[] = [
  {
    path: "setting",
    name: SETTING_ROUTE_WORKSPACE,
    meta: { title: () => t("common.settings") },
    components: {
      content: () => import("@/layouts/SettingLayout.vue"),
      leftSidebar: SettingSidebar,
    },
    props: {
      content: true,
      leftSidebar: true,
    },
    children: [
      {
        path: "profile",
        name: SETTING_ROUTE_PROFILE,
        meta: { title: () => t("settings.sidebar.profile") },
        component: () => import("@/views/ProfileDashboard.vue"),
        props: true,
      },
      {
        path: "profile/two-factor",
        name: SETTING_ROUTE_PROFILE_TWO_FACTOR,
        meta: { title: () => t("two-factor.self") },
        component: () => import("@/views/TwoFactorSetup.vue"),
        props: true,
      },
      {
        path: "general",
        name: SETTING_ROUTE_WORKSPACE_GENERAL,
        meta: {
          title: () => t("settings.sidebar.general"),
          requiredWorkspacePermissionList: () => ["bb.settings.get"],
        },
        component: () => import("@/views/SettingWorkspaceGeneral.vue"),
        props: true,
      },
      {
        path: "member",
        name: SETTING_ROUTE_WORKSPACE_MEMBER,
        meta: {
          title: () => t("settings.sidebar.members"),
          requiredWorkspacePermissionList: () => ["bb.policies.get"],
        },
        component: () => import("@/views/SettingWorkspaceMember.vue"),
        props: true,
      },
      {
        path: "role",
        name: SETTING_ROUTE_WORKSPACE_ROLE,
        meta: {
          title: () => t("settings.sidebar.custom-roles"),
          requiredWorkspacePermissionList: () => ["bb.roles.list"],
        },
        component: () => import("@/views/SettingWorkspaceRole.vue"),
        props: true,
      },
      {
        path: "subscription",
        name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
        meta: {
          title: () => t("settings.sidebar.subscription"),
          requiredWorkspacePermissionList: () => ["bb.settings.get"],
        },
        component: () => import("@/views/SettingWorkspaceSubscription.vue"),
        props: true,
      },
      {
        path: "debug-log",
        name: SETTING_ROUTE_WORKSPACE_DEBUG_LOG,
        meta: {
          title: () => t("settings.sidebar.debug-log"),
          requiredWorkspacePermissionList: () => ["bb.settings.get"],
        },
        component: () => import("@/views/SettingWorkspaceDebugLog.vue"),
        props: true,
      },
      {
        path: "archive",
        name: SETTING_ROUTE_WORKSPACE_ARCHIVE,
        meta: {
          title: () => t("common.archived"),
          requiredWorkspacePermissionList: () => ["bb.settings.get"],
        },
        component: () => import("@/views/Archive.vue"),
        props: true,
      },
    ],
  },
];

export default workspaceSettingRoutes;
