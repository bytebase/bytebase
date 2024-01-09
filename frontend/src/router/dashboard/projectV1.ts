import { RouteRecordRaw } from "vue-router";
import ProjectSidebar from "@/components/Project/ProjectSidebar.vue";

const projectV1Routes: RouteRecordRaw[] = [
  {
    path: "projects/:projectId",
    components: {
      content: () => import("@/layouts/ProjectV1Layout.vue"),
      leftSidebar: ProjectSidebar,
    },
    props: { content: true, leftSidebar: true },
    children: [
      {
        path: "databases",
        name: "workspace.project.database.dashboard",
        meta: {
          overrideTitle: true,
        },
        component: () => import("@/views/project/ProjectDatabaseDashboard.vue"),
        props: true,
      },
      {
        path: "issues",
        name: "workspace.project.issue.dashboard",
        meta: {
          overrideTitle: true,
        },
        component: () => import("@/views/project/ProjectIssueDashboard.vue"),
        props: true,
      },
    ],
  },
];

export default projectV1Routes;
