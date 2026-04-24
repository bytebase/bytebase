import { useSyncExternalStore } from "react";
import type { RouteLocationRaw } from "vue-router";
import { router } from "@/router";
import { AUTH_SIGNIN_MODULE } from "@/router/auth";
import {
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DETAIL,
} from "@/router/dashboard/projectV1";
import {
  WORKSPACE_ROUTE_LANDING,
  WORKSPACE_ROUTE_MY_ISSUES,
} from "@/router/dashboard/workspaceRoutes";
import {
  SETTING_ROUTE_PROFILE,
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
} from "@/router/dashboard/workspaceSetting";
import {
  SQL_EDITOR_DATABASE_MODULE,
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_PROJECT_MODULE,
} from "@/router/sqlEditor";

export {
  AUTH_SIGNIN_MODULE,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DETAIL,
  SETTING_ROUTE_PROFILE,
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
  SQL_EDITOR_DATABASE_MODULE,
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_PROJECT_MODULE,
  WORKSPACE_ROUTE_LANDING,
  WORKSPACE_ROUTE_MY_ISSUES,
};

export type ReactRoute = {
  name?: string;
  fullPath: string;
  params: Record<string, string | string[] | undefined>;
  query: Record<string, unknown>;
};

export type ReactResolvedRoute = {
  href: string;
  fullPath: string;
};

let cachedRoute: ReactRoute | undefined;

function snapshotRoute(): ReactRoute {
  const route = router.currentRoute.value;
  if (cachedRoute?.fullPath === route.fullPath) {
    return cachedRoute;
  }
  cachedRoute = {
    name: route.name?.toString(),
    fullPath: route.fullPath,
    params: route.params as Record<string, string | string[] | undefined>,
    query: route.query as Record<string, unknown>,
  };
  return cachedRoute;
}

function subscribeRoute(onStoreChange: () => void) {
  return router.afterEach(() => onStoreChange());
}

export function useCurrentRoute(): ReactRoute {
  return useSyncExternalStore(subscribeRoute, snapshotRoute, snapshotRoute);
}

export function resolveRoute(to: RouteLocationRaw): ReactResolvedRoute {
  const route = router.resolve(to);
  return {
    href: route.href,
    fullPath: route.fullPath,
  };
}

export function useNavigate() {
  return {
    push: (to: RouteLocationRaw) => router.push(to),
    replace: (to: RouteLocationRaw) => router.replace(to),
    resolve: resolveRoute,
  };
}

export function isSqlEditorRouteName(name: string | undefined): boolean {
  return name?.startsWith("sql-editor") ?? false;
}
