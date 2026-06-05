import {
  useLocation,
  useNavigate as useReactRouterNavigate,
  useMatches,
  useParams,
} from "react-router-dom";
import type { Permission } from "@/types";
import type { RouterLocation, RouterMatch } from "./navigation";
import {
  buildSearchString,
  getAppRouterState,
  getRegisteredRoutes,
  isAppRouterReady,
  navigateByName,
  navigateToPath,
  resolvePath,
  routerGo,
  subscribeRoute,
} from "./navigation";

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
  hash: string;
  params: Record<string, string | string[] | undefined>;
  query: Record<string, unknown>;
  requiredPermissions: Permission[];
  title?: string;
  overrideDocumentTitle: boolean;
  // Mirrors vue-router `route.meta` — carries the per-route handle so legacy
  // `currentRoute.value.meta.*` reads keep working.
  meta: Record<string, unknown>;
};

// vue-router-style navigation target (kept for source compatibility with the
// ~existing consumers): a raw path string, or `{ name, params, query }`.
export type RouteTarget =
  | string
  | {
      name?: string;
      path?: string;
      // Pre-resolved full path (e.g. the result of `router.resolve(...)` or a
      // spread `currentRoute.value`); used when no `name`/`path` is given.
      fullPath?: string;
      params?: Record<string, string | string[] | undefined>;
      query?: Record<string, unknown>;
      hash?: string;
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
  const hash = to.hash
    ? to.hash.startsWith("#")
      ? to.hash
      : `#${to.hash}`
    : "";
  if (to.name) {
    return `${resolvePath(to.name, {
      params: inheritCurrentParams(to.params),
      query: to.query,
    })}${hash}`;
  }
  if (to.path) {
    const search = to.query ? buildSearchString(to.query) : "";
    return `${search ? `${to.path}?${search}` : to.path}${hash}`;
  }
  // Pre-resolved full path (already carries its own query/hash).
  if (to.fullPath) {
    return to.fullPath;
  }
  // No name/path/fullPath: a query (and/or hash) update against the *current*
  // location. vue-router's `router.replace({ query })` keeps the current path;
  // returning "/" here instead would bounce the user to the workspace root
  // (and its redirect), e.g. the DatabasesPage URL-sync landing on /issues.
  const search = to.query ? buildSearchString(to.query) : "";
  const path = window.location.pathname;
  return `${search ? `${path}?${search}` : path}${hash}`;
}

function inheritCurrentParams(
  params: Record<string, string | string[] | undefined> | undefined
): Record<string, string | string[] | undefined> {
  const merged = { ...currentRouteSnapshot().params };
  for (const [key, value] of Object.entries(params ?? {})) {
    if (value !== undefined) {
      merged[key] = value;
    }
  }
  return merged;
}

// Shared builder for both the `useCurrentRoute` hook and the non-hook
// `router.currentRoute.value` snapshot, so the two stay shape-identical.
// Exported as `buildReactRoute` for the app root's leave-guard blocker, which
// matches `nextLocation` against the route table (matchRoutes lives in the
// `.tsx` layer — this module must not import the route table).
export function buildReactRoute(
  location: Pick<RouterLocation, "pathname" | "search" | "hash">,
  matches: ReadonlyArray<{ handle?: unknown }>,
  params: Record<string, string | string[] | undefined>
): ReactRoute {
  return assembleRoute(location, matches, params);
}

function assembleRoute(
  location: Pick<RouterLocation, "pathname" | "search" | "hash">,
  matches: ReadonlyArray<{ handle?: unknown }>,
  params: Record<string, string | string[] | undefined>
): ReactRoute {
  const leafHandle = matches.at(-1)?.handle as RouteHandle | undefined;
  const route: ReactRoute = {
    name: leafHandle?.name,
    fullPath: `${location.pathname}${location.search}${location.hash}`,
    hash: location.hash,
    params,
    query: Object.fromEntries(new URLSearchParams(location.search)),
    requiredPermissions: dedupePermissions(
      matches.flatMap(
        (m) =>
          (m.handle as RouteHandle | undefined)?.requiredPermissionList?.() ??
          []
      )
    ),
    overrideDocumentTitle: leafHandle?.overrideDocumentTitle ?? false,
    meta: (leafHandle as Record<string, unknown> | undefined) ?? {},
  };
  route.title = leafHandle?.title?.(route);
  return route;
}

/** React-router-backed current route, shaped like the legacy bridge. */
export function useCurrentRoute(): ReactRoute {
  const location = useLocation();
  const params = useParams();
  const matches = useMatches();
  return assembleRoute(
    location,
    matches,
    params as Record<string, string | string[] | undefined>
  );
}

// Non-hook snapshot of the current route, read from the registered data router
// (backs `router.currentRoute.value`).
//
// Memoized so it returns a referentially STABLE object between actual route
// changes. `router.currentRoute.value` backs the `useReactiveRoute`
// `useSyncExternalStore` snapshot (and ~110 imperative call sites); a fresh
// object on every read makes external stores re-render forever ("The result of
// getSnapshot should be cached to avoid an infinite loop"). `fullPath` + `name`
// fully identify the route — params, query and matches all derive from the URL.
let currentRouteCache: ReactRoute | undefined;
function currentRouteSnapshot(): ReactRoute {
  const state = getAppRouterState();
  const location = state?.location ?? { pathname: "/", search: "", hash: "" };
  const matches: RouterMatch[] = state?.matches ?? [];
  const leafParams = (matches.at(-1)?.params ?? {}) as Record<
    string,
    string | string[] | undefined
  >;
  const next = assembleRoute(location, matches, leafParams);
  if (
    !currentRouteCache ||
    currentRouteCache.fullPath !== next.fullPath ||
    currentRouteCache.name !== next.name
  ) {
    currentRouteCache = next;
  }
  return currentRouteCache;
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

// --- vue-router-instance drop-in --------------------------------------------
// A module-level object mirroring the imperative surface of the legacy
// vue-router instance (`import { router } from "@/router"`), backed by the
// react-router data router. Lets the ~110 imperative call sites keep their
// usage unchanged through teardown — only the import path moves to
// `@/react/router`.

// vue-router `next()` callback: no-arg / `RouteTarget` proceeds, `false`
// cancels.
type GuardNext = (target?: boolean | RouteTarget) => void;
export type NavigationHistoryAction = "POP" | "PUSH" | "REPLACE";
export type BeforeEachGuardOptions = {
  historyAction?: NavigationHistoryAction;
  reset?: () => void;
  retry?: () => void;
};
type BeforeEachGuard = (
  to: ReactRoute,
  from: ReactRoute,
  next: GuardNext,
  options?: BeforeEachGuardOptions
) => void;

const beforeEachGuards = new Set<BeforeEachGuard>();

// Runs every registered `beforeEach` guard against a pending navigation and
// reports whether it should be BLOCKED (a guard called `next(false)`). The app
// root consults this from a single `useBlocker`, reproducing vue-router's
// global guard semantics. Guards that redirect (`next(target)`) are treated as
// "proceed" here — the existing leave guards only ever proceed or cancel.
export function runBeforeEachGuards(
  to: ReactRoute,
  from: ReactRoute,
  options?: BeforeEachGuardOptions
): boolean {
  for (const guard of beforeEachGuards) {
    let blocked = false;
    guard(
      to,
      from,
      (target) => {
        if (target === false) blocked = true;
      },
      options
    );
    if (blocked) return true;
  }
  return false;
}

export const router = {
  push: (to: RouteTarget) => navigateToPath(resolveTarget(to)),
  replace: (to: RouteTarget) =>
    navigateToPath(resolveTarget(to), { replace: true }),
  resolve: (to: RouteTarget): ReactResolvedRoute => resolveRoute(to),
  back: () => routerGo(-1),
  go: (delta: number) => routerGo(delta),
  isReady: () => isAppRouterReady(),
  // vue-router exposed `currentRoute` as a Ref; consumers read `.value`.
  get currentRoute(): { value: ReactRoute } {
    return {
      get value() {
        return currentRouteSnapshot();
      },
    };
  },
  beforeEach(guard: BeforeEachGuard): () => void {
    beforeEachGuards.add(guard);
    return () => {
      beforeEachGuards.delete(guard);
    };
  },
  afterEach(hook: () => void): () => void {
    return subscribeRoute(hook);
  },
  // Named routes for the agent's route-map listing; shaped like the vue-router
  // records the agent reads (`path` / `name` / `children`).
  getRoutes(): { path: string; name?: string; children: unknown[] }[] {
    return getRegisteredRoutes().map((r) => ({
      path: r.path,
      name: r.name,
      children: [],
    }));
  },
};

// The drop-in router's type, for code that takes the router as a parameter
// (e.g. the agent tool factories) instead of importing it directly.
export type AppRouterInstance = typeof router;

// Re-exported so non-hook callers can navigate by name.
export { navigateByName };
