import type { RouteRecordRaw } from "vue-router";
import InstanceLayout from "@/layouts/InstanceLayout.vue";
import { t } from "@/plugins/i18n";
import { INSTANCE_ROUTE_DASHBOARD } from "./workspaceRoutes";

export const INSTANCE_ROUTE_CREATE = `${INSTANCE_ROUTE_DASHBOARD}.create`;
export const INSTANCE_ROUTE_DETAIL = `${INSTANCE_ROUTE_DASHBOARD}.detail`;
export const INSTANCE_ROUTE_DATABASE_DETAIL = `${INSTANCE_ROUTE_DASHBOARD}.database.detail`;

const instanceRoutes: RouteRecordRaw[] = [
  {
    path: "instances/new",
    name: INSTANCE_ROUTE_CREATE,
    meta: {
      title: () => t("instance.new-instance"),
      getQuickActionList: () => [],
    },
    components: {
      content: () => import("@/views/CreateInstancePage.vue"),
      leftSidebar: () => import("@/views/DashboardSidebar.vue"),
    },
    props: { content: true },
  },
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
