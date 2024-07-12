import type { RouteLocationNormalized, RouteRecordRaw } from "vue-router";
import { useEnvironmentV1Store } from "@/store";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import { unknownEnvironment } from "@/types";
import DashboardSidebar from "@/views/DashboardSidebar.vue";
import { ENVIRONMENT_V1_ROUTE_DASHBOARD } from "./workspaceRoutes";

export const ENVIRONMENT_V1_ROUTE_DETAIL = `${ENVIRONMENT_V1_ROUTE_DASHBOARD}.detail`;

const environmentV1Routes: RouteRecordRaw[] = [
  {
    path: "environments/:environmentName",
    name: ENVIRONMENT_V1_ROUTE_DETAIL,
    meta: {
      title: (route: RouteLocationNormalized) => {
        const environmentName = route.params.environmentName as string;
        return (
          useEnvironmentV1Store().getEnvironmentByName(
            `${environmentNamePrefix}${environmentName}`
          ) || unknownEnvironment()
        ).title;
      },
    },
    components: {
      content: () => import("@/views/EnvironmentDetail.vue"),
      leftSidebar: DashboardSidebar,
    },
    props: { content: true },
  },
];

export default environmentV1Routes;
