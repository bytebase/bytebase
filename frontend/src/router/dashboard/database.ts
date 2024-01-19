import { RouteRecordRaw } from "vue-router";
import { t } from "@/plugins/i18n";
import DashboardSidebar from "@/views/DashboardSidebar.vue";
import { DATABASE_ROUTE_DASHBOARD } from "./workspaceRoutes";

const databaseRoutes: RouteRecordRaw[] = [
  {
    path: "db",
    name: DATABASE_ROUTE_DASHBOARD,
    meta: {
      title: () => t("common.databases"),
      getQuickActionList: () => {
        return ["quickaction.bb.database.create"];
      },
      // Workspace-level database list is accessible to all users.
      requiredProjectPermissionList: () => [],
    },
    components: {
      content: () => import("@/views/DatabaseDashboard.vue"),
      leftSidebar: DashboardSidebar,
    },
    props: { content: true, leftSidebar: true },
  },
];

export default databaseRoutes;
