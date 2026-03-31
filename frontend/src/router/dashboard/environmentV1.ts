import type { RouteRecordRaw } from "vue-router";
import { t } from "@/plugins/i18n";
import { ENVIRONMENT_V1_ROUTE_DASHBOARD } from "./workspaceRoutes";

export const ENVIRONMENT_V1_ROUTE_DETAIL = `${ENVIRONMENT_V1_ROUTE_DASHBOARD}.detail`;

const environmentV1Routes: RouteRecordRaw[] = [
  {
    path: "environments/:environmentName",
    name: ENVIRONMENT_V1_ROUTE_DETAIL,
    meta: {
      title: () => t("common.environment"),
      requiredPermissionList: () => ["bb.settings.getEnvironment"],
    },
    redirect: (to) => ({
      name: ENVIRONMENT_V1_ROUTE_DASHBOARD,
      hash: `#${to.params.environmentName}`,
    }),
  },
];

export default environmentV1Routes;
