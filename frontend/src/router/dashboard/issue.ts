import { RouteRecordRaw } from "vue-router";
import ProjectSidebar from "@/components/Project/ProjectSidebar.vue";
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
      leftSidebar: ProjectSidebar,
    },
    props: { content: true, leftSidebar: true },
  },
];

export default issueRoutes;
