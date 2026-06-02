import type { RouteLocationNormalized, RouteRecordRaw } from "vue-router";
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
      requiredPermissionList: () => ["bb.instances.create"],
      getQuickActionList: () => [],
    },
    components: {
      content: () => import("@/react/ReactPageMount.vue"),
      leftSidebar: () => import("@/react/ReactSidebarMount.vue"),
    },
    props: {
      content: () => ({ page: "CreateInstancePage" }),
    },
  },
  {
    path: "instances/:instanceId",
    components: {
      content: () => import("@/react/ReactRouteShellBridge.vue"),
      leftSidebar: () => import("@/react/ReactSidebarMount.vue"),
    },
    meta: {
      title: () => t("common.instance"),
    },
    props: {
      content: (route: RouteLocationNormalized) => ({
        page: "InstanceRouteShell",
        pageProps: {
          instanceId: route.params.instanceId,
        },
      }),
    },
    children: [
      {
        path: "",
        name: INSTANCE_ROUTE_DETAIL,
        meta: {
          requiredPermissionList: () => ["bb.instances.get"],
        },
        component: () => import("@/react/ReactPageMount.vue"),
        props: (route: RouteLocationNormalized) => ({
          page: "InstanceDetailPage",
          instanceId: route.params.instanceId,
        }),
      },
      {
        path: "databases/:databaseName",
        name: INSTANCE_ROUTE_DATABASE_DETAIL,
        meta: {
          requiredPermissionList: () => ["bb.projects.get", "bb.databases.get"],
        },
        component: () => import("@/react/ReactPageMount.vue"),
        props: (route: RouteLocationNormalized) => ({
          page: "InstanceDatabaseRedirectPage",
          instanceId: route.params.instanceId,
          databaseName: route.params.databaseName,
        }),
      },
    ],
  },
];

export default instanceRoutes;
