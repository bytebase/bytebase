import type { RouteRecordRaw } from "vue-router";
import { t } from "@/plugins/i18n";
import DashboardSidebar from "@/views/DashboardSidebar.vue";

export const ISSUE_ROUTE_DASHBOARD = "workspace.issue";
export const ISSUE_ROUTE_DETAIL = `${ISSUE_ROUTE_DASHBOARD}.detail`;

const issueRoutes: RouteRecordRaw[] = [
  {
    path: "issue",
    name: ISSUE_ROUTE_DASHBOARD,
    meta: {
      title: () => t("common.issues"),
    },
    components: {
      content: () => import("@/views/IssueDashboard.vue"),
      leftSidebar: DashboardSidebar,
    },
    props: { content: true, leftSidebar: true },
  },
];

export default issueRoutes;
