import type { RouteRecordRaw } from "vue-router";
import InstanceLayout from "@/layouts/InstanceLayout.vue";
import { t } from "@/plugins/i18n";
import { INSTANCE_ROUTE_DASHBOARD } from "./workspaceRoutes";

export const INSTANCE_ROUTE_DETAIL = `${INSTANCE_ROUTE_DASHBOARD}.detail`;
export const INSTANCE_ROUTE_DATABASE_DETAIL = `${INSTANCE_ROUTE_DASHBOARD}.database.detail`;

const instanceRoutes: RouteRecordRaw[] = [
  {
    path: "instances/:instanceId",
    components: {
      content: InstanceLayout,
      leftSidebar: () => import("@/views/DashboardSidebar.vue"),
    },
    meta: {
      title: () => t("common.instance"),
    },
    props: { content: true },
    children: [
      {
        path: "",
        name: INSTANCE_ROUTE_DETAIL,
        meta: {
          requiredPermissionList: () => ["bb.instances.get"],
        },
        component: () => import("@/views/InstanceDetail.vue"),
        props: true,
      },
      {
        path: "databases/:databaseName",
        name: INSTANCE_ROUTE_DATABASE_DETAIL,
        meta: {
          requiredPermissionList: () => ["bb.projects.get", "bb.databases.get"],
        },
        component: () => import("@/components/InstanceDatabaseRedirect.vue"),
        props: true,
      },
    ],
  },
];

export default instanceRoutes;
