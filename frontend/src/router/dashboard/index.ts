import { RouteRecordRaw } from "vue-router";
import BodyLayout from "@/layouts/BodyLayout.vue";
import DashboardLayout from "@/layouts/DashboardLayout.vue";
import databaseRoutes from "./database";
import environmentRoutes from "./environment";
import instanceRoutes from "./instance";
import issueRoutes from "./issue";
import projectV1Routes from "./projectV1";
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
          ...projectV1Routes,
          ...issueRoutes,
          ...databaseRoutes,
        ],
      },
    ],
  },
];

export default dashboardRoutes;
