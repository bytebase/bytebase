import { RouteLocationNormalized, RouteRecordRaw } from "vue-router";
import { t } from "@/plugins/i18n";
import { useEnvironmentV1Store } from "@/store";
import { idFromSlug } from "@/utils";
import DashboardSidebar from "@/views/DashboardSidebar.vue";

export const ENVIRONMENT_ROUTE_DASHBOARD = "workspace.environment";
export const ENVIRONMENT_ROUTE_DETAIL = `${ENVIRONMENT_ROUTE_DASHBOARD}.detail`;

const environmentRoutes: RouteRecordRaw[] = [
  {
    path: "environment",
    name: ENVIRONMENT_ROUTE_DASHBOARD,
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
    path: "environment/:environmentSlug",
    name: ENVIRONMENT_ROUTE_DETAIL,
    meta: {
      title: (route: RouteLocationNormalized) => {
        const slug = route.params.environmentSlug as string;
        return useEnvironmentV1Store().getEnvironmentByUID(
          String(idFromSlug(slug))
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

export default environmentRoutes;
