import { RouteRecordRaw } from "vue-router";
import { t } from "@/plugins/i18n";
import DashboardSidebar from "@/views/DashboardSidebar.vue";
import { DATABASE_ROUTE_DASHBOARD } from "./workspaceRoutes";

export const DATABASE_ROUTE_DETAIL = `${DATABASE_ROUTE_DASHBOARD}.detail`;
export const DATABASE_ROUTE_CHANGE_HISTORY_DETAIL = `${DATABASE_ROUTE_DASHBOARD}.change-history.detail`;

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
  {
    path: "projects/:projectId/instances/:instanceId/databases/:databaseName",
    components: {
      content: () => import("@/layouts/DatabaseLayout.vue"),
      leftSidebar: () => import("@/components/Project/ProjectSidebarV1.vue"),
    },
    props: { content: true, leftSidebar: true },
    children: [
      {
        path: "",
        name: DATABASE_ROUTE_DETAIL,
        meta: {
          overrideTitle: true,
          requiredProjectPermissionList: () => [
            "bb.projects.get",
            "bb.databases.get",
          ],
        },
        component: () => import("@/views/DatabaseDetail.vue"),
        props: true,
      },
      {
        path: "change-histories/:changeHistoryId",
        name: DATABASE_ROUTE_CHANGE_HISTORY_DETAIL,
        meta: {
          overrideTitle: true,
          requiredProjectPermissionList: () => [
            "bb.projects.get",
            "bb.databases.get",
            "bb.changeHistories.get",
          ],
        },
        component: () => import("@/views/ChangeHistoryDetail.vue"),
        props: true,
      },
    ],
  },
];

export default databaseRoutes;
