import { RouteLocationNormalized, RouteRecordRaw } from "vue-router";
import { t } from "@/plugins/i18n";
import { useEnvironmentV1Store } from "@/store";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import { unknownEnvironment } from "@/types";
import DashboardSidebar from "@/views/DashboardSidebar.vue";
import { ENVIRONMENT_V1_ROUTE_DASHBOARD } from "./workspaceRoutes";

export const ENVIRONMENT_V1_ROUTE_DETAIL = `${ENVIRONMENT_V1_ROUTE_DASHBOARD}.detail`;

const environmentV1Routes: RouteRecordRaw[] = [
  {
    path: "environments",
    name: ENVIRONMENT_V1_ROUTE_DASHBOARD,
    meta: {
      title: () => t("common.environments"),
      getQuickActionList: () => {
        return [
          "quickaction.bb.environment.create",
          "quickaction.bb.environment.reorder",
        ];
      },
    },
    components: {
      content: () => import("@/views/EnvironmentDashboard.vue"),
      leftSidebar: DashboardSidebar,
    },
    props: { content: true, leftSidebar: true },
  },
  {
    path: "environments/:environmentId",
    name: ENVIRONMENT_V1_ROUTE_DETAIL,
    meta: {
      title: (route: RouteLocationNormalized) => {
        const environmentId = route.params.environmentId as string;
        return (
          useEnvironmentV1Store().getEnvironmentByName(
            `${environmentNamePrefix}${environmentId}`
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
