import type { RouteRecordRaw } from "vue-router";
import DashboardSidebar from "@/views/DashboardSidebar.vue";
import { ENVIRONMENT_V1_ROUTE_DASHBOARD } from "./workspaceRoutes";

export const ENVIRONMENT_V1_ROUTE_DETAIL = `${ENVIRONMENT_V1_ROUTE_DASHBOARD}.detail`;

const environmentV1Routes: RouteRecordRaw[] = [
  {
    path: "environments/:environmentName",
    name: ENVIRONMENT_V1_ROUTE_DETAIL,
    components: {
      content: () => import("@/views/EnvironmentDetail.vue"),
      leftSidebar: DashboardSidebar,
    },
    props: { content: true },
  },
];

export default environmentV1Routes;
