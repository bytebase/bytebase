import { startCase } from "lodash-es";
import { RouteRecordRaw } from "vue-router";
import { t } from "@/plugins/i18n";
import DashboardSidebar from "@/views/DashboardSidebar.vue";
import Home from "@/views/Home.vue";
import SettingSidebar from "@/views/SettingSidebar.vue";
import {
  WORKSPACE_HOME_MODULE,
  WORKSPACE_ROUTE_SLOW_QUERY,
  WORKSPACE_ROUTE_EXPORT_CENTER,
  WORKSPACE_ROUTE_ANOMALY_CENTER,
  WORKSPACE_ROUTE_USER_PROFILE,
} from "./workspaceRoutes";

const workspaceRoutes: RouteRecordRaw[] = [
  {
    path: "",
    name: WORKSPACE_HOME_MODULE,
    meta: {
      getQuickActionList: () => {
        return [
          "quickaction.bb.database.schema.update",
          "quickaction.bb.database.data.update",
          "quickaction.bb.database.create",
          "quickaction.bb.instance.create",
          "quickaction.bb.issue.grant.request.querier",
          "quickaction.bb.issue.grant.request.exporter",
        ];
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
    path: "slow-query",
    name: WORKSPACE_ROUTE_SLOW_QUERY,
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
    name: WORKSPACE_ROUTE_EXPORT_CENTER,
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
    name: WORKSPACE_ROUTE_ANOMALY_CENTER,
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
    name: WORKSPACE_ROUTE_USER_PROFILE,
    components: {
      content: () => import("@/views/ProfileDashboard.vue"),
      leftSidebar: SettingSidebar,
    },
    props: true,
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
];

export default workspaceRoutes;
