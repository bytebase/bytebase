import type { RouteLocationNormalized, RouteRecordRaw } from "vue-router";
import DummyRootView from "@/DummyRootView";
import { t } from "@/plugins/i18n";
import {
  databaseNamePrefix,
  instanceNamePrefix,
  pushNotification,
  useDatabaseV1Store,
} from "@/store";
import { isValidProjectName } from "@/types";
import { extractProjectResourceName } from "@/utils/v1";
import { PROJECT_V1_ROUTE_DATABASE_DETAIL } from "./projectV1";
import { INSTANCE_ROUTE_DASHBOARD } from "./workspaceRoutes";

export const INSTANCE_ROUTE_CREATE = `${INSTANCE_ROUTE_DASHBOARD}.create`;
export const INSTANCE_ROUTE_DETAIL = `${INSTANCE_ROUTE_DASHBOARD}.detail`;
export const INSTANCE_ROUTE_DATABASE_DETAIL = `${INSTANCE_ROUTE_DASHBOARD}.database.detail`;

const redirectToProjectDatabaseDetail = async (
  route: RouteLocationNormalized
) => {
  const instanceId = route.params.instanceId as string;
  const databaseName = route.params.databaseName as string;
  try {
    const database = await useDatabaseV1Store().getOrFetchDatabaseByName(
      `${instanceNamePrefix}${instanceId}/${databaseNamePrefix}${databaseName}`
    );
    if (database && isValidProjectName(database.project)) {
      return {
        name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
        params: {
          projectId: extractProjectResourceName(database.project),
          instanceId,
          databaseName,
        },
      };
    }
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: "Database not found",
      description: `Database: ${databaseName}`,
    });
  } catch (error) {
    console.error("Failed to fetch database:", error);
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Error",
      description: `Failed to load database: ${databaseName}`,
    });
  }
  return {
    name: INSTANCE_ROUTE_DETAIL,
    params: {
      instanceId,
    },
  };
};

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
        component: DummyRootView,
        beforeEnter: redirectToProjectDatabaseDetail,
      },
    ],
  },
];

export default instanceRoutes;
