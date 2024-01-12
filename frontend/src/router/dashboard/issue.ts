import { RouteRecordRaw } from "vue-router";
import ProjectSidebarV1 from "@/components/Project/ProjectSidebarV1.vue";
import { t } from "@/plugins/i18n";
import DashboardSidebar from "@/views/DashboardSidebar.vue";

const issueRoutes: RouteRecordRaw[] = [
  {
    path: "issue",
    name: "workspace.issue",
    meta: {
      title: () => t("common.issues"),
    },
    components: {
      content: () => import("@/views/IssueDashboard.vue"),
      leftSidebar: DashboardSidebar,
    },
    props: { content: true, leftSidebar: true },
  },
  {
    path: "issue/:issueSlug",
    name: "workspace.issue.detail",
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
