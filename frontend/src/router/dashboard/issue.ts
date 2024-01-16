import { RouteRecordRaw } from "vue-router";
import ProjectSidebarV1 from "@/components/Project/ProjectSidebarV1.vue";
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
  // legacy issue detail route have to be kept for a long time.
  {
    path: "issue/:issueSlug",
    name: ISSUE_ROUTE_DETAIL,
    meta: {
      overrideTitle: true,
    },
    components: {
      content: () => import("@/views/IssueDetailV1.vue"),
      leftSidebar: ProjectSidebarV1,
    },
    props: { content: true, leftSidebar: true },
  },
];

export default issueRoutes;
