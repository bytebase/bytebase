import { RouteRecordRaw } from "vue-router";
import BodyLayout from "@/layouts/BodyLayout.vue";
import DashboardLayout from "@/layouts/DashboardLayout.vue";
import databaseRoutes from "./database";
import environmentRoutes from "./environment";
import instanceRoutes from "./instance";
import issueRoutes from "./issue";
import projectRoutes from "./project";
import projectV1Routes from "./projectv1";
import workspaceRoutes from "./workspace";
import workspaceSettingRoutes from "./workspaceSetting";

const dashboardRoutes: RouteRecordRaw[] = [
  {
    path: "/",
    component: DashboardLayout,
    children: [
      {
        path: "",
        components: { body: BodyLayout },
        children: [
          ...workspaceRoutes,
          ...workspaceSettingRoutes,
          ...environmentRoutes,
          ...instanceRoutes,
          ...projectRoutes,
          ...projectV1Routes,
          ...issueRoutes,
          ...databaseRoutes,
        ],
      },
    ],
  },
];

export default dashboardRoutes;
