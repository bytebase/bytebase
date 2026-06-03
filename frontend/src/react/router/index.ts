import {
  useLocation,
  useNavigate as useReactRouterNavigate,
  useMatches,
  useParams,
} from "react-router-dom";
import type { Permission } from "@/types";
import { navigateByName, navigateToPath, resolvePath } from "./navigation";

// Re-export the route-name constants from the vue-free handles module so React
// consumers keep importing them from `@/react/router`.
export {
  AUTH_SIGNIN_MODULE,
  DATABASE_ROUTE_DASHBOARD,
  ENVIRONMENT_V1_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_DETAIL,
  SETTING_ROUTE_PROFILE,
  SETTING_ROUTE_WORKSPACE_GENERAL,
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
  SQL_EDITOR_DATABASE_MODULE,
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_PROJECT_MODULE,
  WORKSPACE_ROUTE_AUDIT_LOG,
  WORKSPACE_ROUTE_CUSTOM_APPROVAL,
  WORKSPACE_ROUTE_DATA_CLASSIFICATION,
  WORKSPACE_ROUTE_GLOBAL_MASKING,
  WORKSPACE_ROUTE_GROUPS,
  WORKSPACE_ROUTE_IDENTITY_PROVIDERS,
  WORKSPACE_ROUTE_IM,
  WORKSPACE_ROUTE_LANDING,
  WORKSPACE_ROUTE_MCP,
  WORKSPACE_ROUTE_MEMBERS,
  WORKSPACE_ROUTE_MY_ISSUES,
  WORKSPACE_ROUTE_RISK_ASSESSMENT,
  WORKSPACE_ROUTE_ROLES,
  WORKSPACE_ROUTE_SEMANTIC_TYPES,
  WORKSPACE_ROUTE_SERVICE_ACCOUNTS,
  WORKSPACE_ROUTE_SQL_REVIEW,
  WORKSPACE_ROUTE_USER_PROFILE,
  WORKSPACE_ROUTE_USERS,
  WORKSPACE_ROUTE_WORKLOAD_IDENTITIES,
} from "./handles";

export type ReactRoute = {
  name?: string;
  fullPath: string;
  params: Record<string, string | string[] | undefined>;
  query: Record<string, unknown>;
  requiredPermissions: Permission[];
  title?: string;
  overrideDocumentTitle: boolean;
};

// vue-router-style navigation target (kept for source compatibility with the
// ~existing consumers): a raw path string, or `{ name, params, query }`.
export type RouteTarget =
  | string
  | {
      name?: string;
      path?: string;
      params?: Record<string, string>;
      query?: Record<string, string | undefined>;
    };

export type ReactResolvedRoute = { href: string; fullPath: string };

// Per-route metadata carried on `handle` (mirrors the legacy vue-router
// `meta`). Ported route definitions attach these alongside `name`.
type RouteHandle = {
  name?: string;
  requiredPermissionList?: () => Permission[];
  title?: (route: ReactRoute) => string | undefined;
  overrideDocumentTitle?: boolean;
};

function dedupePermissions(permissions: Permission[]): Permission[] {
  return [...new Set(permissions)];
}

function resolveTarget(to: RouteTarget): string {
  if (typeof to === "string") return to;
  if (to.path) {
    const search =
      to.query &&
      new URLSearchParams(
        Object.entries(to.query).filter(
          (e): e is [string, string] => e[1] !== undefined
        )
      ).toString();
    return search ? `${to.path}?${search}` : to.path;
  }
  return resolvePath(to.name ?? "", { params: to.params, query: to.query });
}

/** React-router-backed current route, shaped like the legacy bridge. */
export function useCurrentRoute(): ReactRoute {
  const location = useLocation();
  const params = useParams();
  const matches = useMatches();
  const leaf = matches.at(-1);
  const leafHandle = leaf?.handle as RouteHandle | undefined;
  const route: ReactRoute = {
    name: leafHandle?.name,
    fullPath: `${location.pathname}${location.search}${location.hash}`,
    params: params as Record<string, string | string[] | undefined>,
    query: Object.fromEntries(new URLSearchParams(location.search)),
    requiredPermissions: dedupePermissions(
      matches.flatMap(
        (m) =>
          (m.handle as RouteHandle | undefined)?.requiredPermissionList?.() ??
          []
      )
    ),
    overrideDocumentTitle: leafHandle?.overrideDocumentTitle ?? false,
  };
  route.title = leafHandle?.title?.(route);
  return route;
}

export function resolveRoute(to: RouteTarget): ReactResolvedRoute {
  const fullPath = resolveTarget(to);
  return { href: fullPath, fullPath };
}

export function useNavigate() {
  // Hook call keeps React's router context wired; navigation itself goes
  // through the resolver so by-name targets keep working.
  useReactRouterNavigate();
  return {
    push: (to: RouteTarget) => navigateToPath(resolveTarget(to)),
    replace: (to: RouteTarget) =>
      navigateToPath(resolveTarget(to), { replace: true }),
    resolve: resolveRoute,
  };
}

export function isSqlEditorRouteName(name: string | undefined): boolean {
  return name?.startsWith("sql-editor") ?? false;
}

// Re-exported so non-hook callers can navigate by name.
export { navigateByName };
