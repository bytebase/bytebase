import type { RouteRecordRaw } from "vue-router";
import { t } from "@/plugins/i18n";

export const SETTING_ROUTE = "setting";
export const SETTING_ROUTE_WORKSPACE = `${SETTING_ROUTE}.workspace`;
export const SETTING_ROUTE_PROFILE = `${SETTING_ROUTE}.profile`;
export const SETTING_ROUTE_PROFILE_TWO_FACTOR = `${SETTING_ROUTE_PROFILE}.two-factor`;
export const SETTING_ROUTE_WORKSPACE_GENERAL = `${SETTING_ROUTE_WORKSPACE}.general`;
export const SETTING_ROUTE_WORKSPACE_SUBSCRIPTION = `${SETTING_ROUTE_WORKSPACE}.subscription`;
export const SETTING_ROUTE_WORKSPACE_ARCHIVE = `${SETTING_ROUTE_WORKSPACE}.archive`;

const workspaceSettingRoutes: RouteRecordRaw[] = [
  {
    path: "setting",
    name: SETTING_ROUTE_WORKSPACE,
    meta: { title: () => t("common.settings") },
    components: {
      content: () => import("@/layouts/SettingLayout.vue"),
      leftSidebar: () => import("@/views/DashboardSidebar.vue"),
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
          requiredPermissionList: () => [
            "bb.settings.get",
            "bb.settings.getWorkspaceProfile",
            "bb.policies.get",
          ],
        },
        component: () => import("@/views/SettingWorkspaceGeneral.vue"),
        props: true,
      },
      {
        path: "subscription",
        name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
        meta: {
          title: () => t("settings.sidebar.subscription"),
          requiredPermissionList: () => ["bb.settings.get"],
        },
        component: () => import("@/views/SettingWorkspaceSubscription.vue"),
        props: true,
      },
      {
        path: "archive",
        name: SETTING_ROUTE_WORKSPACE_ARCHIVE,
        meta: {
          title: () => t("common.archived"),
          requiredPermissionList: () => [
            "bb.projects.list",
            "bb.instances.list",
          ],
        },
        component: () => import("@/views/Archive.vue"),
        props: true,
      },
    ],
  },
];

export default workspaceSettingRoutes;
