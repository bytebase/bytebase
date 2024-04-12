import type { RouteRecordRaw } from "vue-router";
import BodyLayout from "@/layouts/BodyLayout.vue";
import DashboardLayout from "@/layouts/DashboardLayout.vue";
import databaseRoutes from "./database";
import environmentV1Routes from "./environmentV1";
import instanceRoutes from "./instance";
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
          ...environmentV1Routes,
          ...instanceRoutes,
          ...projectV1Routes,
          ...databaseRoutes,
        ],
      },
    ],
  },
];

export default dashboardRoutes;
