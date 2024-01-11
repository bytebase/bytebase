import { pull, startCase } from "lodash-es";
import { RouteLocationNormalized, RouteRecordRaw } from "vue-router";
import { t } from "@/plugins/i18n";
import { hasFeature, useUserStore } from "@/store";
import { QuickActionType, unknownUser } from "@/types";
import DashboardSidebar from "@/views/DashboardSidebar.vue";
import Home from "@/views/Home.vue";
import SettingSidebar from "@/views/SettingSidebar.vue";

export const WORKSPACE_HOME_MODULE = "workspace.home";

const workspaceRoutes: RouteRecordRaw[] = [
  {
    path: "",
    name: WORKSPACE_HOME_MODULE,
    meta: {
      quickActionListByRole: () => {
        const DBA_AND_OWNER_QUICK_ACTION_LIST: QuickActionType[] = [
          "quickaction.bb.database.schema.update",
          "quickaction.bb.database.data.update",
          "quickaction.bb.database.create",
          "quickaction.bb.instance.create",
        ];
        const DEVELOPER_QUICK_ACTION_LIST: QuickActionType[] = [
          "quickaction.bb.database.schema.update",
          "quickaction.bb.database.data.update",
          "quickaction.bb.issue.grant.request.querier",
          "quickaction.bb.issue.grant.request.exporter",
        ];
        if (hasFeature("bb.feature.dba-workflow")) {
          pull(DEVELOPER_QUICK_ACTION_LIST, "quickaction.bb.database.create");
        }
        return new Map([
          ["OWNER", DBA_AND_OWNER_QUICK_ACTION_LIST],
          ["DBA", DBA_AND_OWNER_QUICK_ACTION_LIST],
          ["DEVELOPER", DEVELOPER_QUICK_ACTION_LIST],
        ]);
      },
    },
    components: {
      content: Home,
      leftSidebar: DashboardSidebar,
    },
    props: {
      content: true,
      leftSidebar: true,
    },
  },
  {
    path: "403",
    name: "error.403",
    components: {
      content: () => import("@/views/Page403.vue"),
      leftSidebar: DashboardSidebar,
    },
    props: {
      content: true,
      leftSidebar: true,
    },
  },
  {
    path: "404",
    name: "error.404",
    components: {
      content: () => import("@/views/Page404.vue"),
      leftSidebar: DashboardSidebar,
    },
    props: {
      content: true,
      leftSidebar: true,
    },
  },
  {
    path: "slow-query",
    name: "workspace.slow-query",
    meta: { title: () => startCase(t("slow-query.slow-queries")) },
    components: {
      content: () => import("@/views/SlowQuery/SlowQueryDashboard.vue"),
      leftSidebar: DashboardSidebar,
    },
    props: {
      content: true,
      leftSidebar: true,
    },
  },
  {
    path: "export-center",
    name: "workspace.export-center",
    meta: { title: () => startCase(t("export-center.self")) },
    components: {
      content: () => import("@/views/ExportCenter/index.vue"),
      leftSidebar: DashboardSidebar,
    },
    props: {
      content: true,
      leftSidebar: true,
    },
  },
  {
    path: "anomaly-center",
    name: "workspace.anomaly-center",
    meta: { title: () => t("anomaly-center") },
    components: {
      content: () => import("@/views/AnomalyCenterDashboard.vue"),
      leftSidebar: DashboardSidebar,
    },
    props: {
      content: true,
      leftSidebar: true,
    },
  },
  {
    // "u" stands for user. Strictly speaking, it's not accurate because we
    // may refer to other principal type in the future. But from the endusers'
    // perspective, they are more familiar with the "user" concept.
    // We make an exception to use a shorthand here because it's a commonly
    // accessed endpoint, and maybe in the future, we will further provide a
    // shortlink such as users/<<email>>
    path: "users/:principalEmail",
    name: "workspace.profile",
    meta: {
      title: (route: RouteLocationNormalized) => {
        const principalEmail = route.params.principalEmail as string;
        const user =
          useUserStore().getUserByEmail(principalEmail) ?? unknownUser();
        return user.title;
      },
    },
    components: {
      content: () => import("@/views/ProfileDashboard.vue"),
      leftSidebar: SettingSidebar,
    },
    props: { content: true },
  },
];

export default workspaceRoutes;
