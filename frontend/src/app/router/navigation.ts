// Structural type for the createBrowserRouter instance (avoids depending on
// @remix-run/router's exported name across react-router versions).
export type RouterMatch = {
  handle?: unknown;
  params: Record<string, string | undefined>;
};
export type RouterLocation = { pathname: string; search: string; hash: string };
export type RouterState = {
  location: RouterLocation;
  matches: RouterMatch[];
  initialized: boolean;
  navigation?: {
    historyAction?: "POP" | "PUSH" | "REPLACE";
  };
  preventScrollReset?: boolean;
};
export type NavigationOptions = {
  replace?: boolean;
  preventScrollReset?: boolean;
};
type AppRouterLike = {
  // Overloaded to match the data router (`navigate(delta)` /
  // `navigate(to, opts)`), so the concrete instance is assignable.
  navigate: {
    (delta: number): Promise<void>;
    (to: string, opts?: NavigationOptions): Promise<void>;
  };
  subscribe?: (fn: (state: RouterState) => void) => () => void;
  state?: RouterState;
};

// react-router navigates by *path*, but the guard and auth lifecycle redirect
// by route *name*, so names resolve to concrete paths via a `name -> path
// pattern` index. The index is BUILT from the route table on the React (.tsx)
// side and REGISTERED here (`setRouteNameIndex`) — this module stays a pure
// `.ts` helper.

type NavParams = Record<string, string | string[] | undefined>;
type NavQuery = Record<string, unknown>;

let nameIndex = new Map<string, string>();

export function setRouteNameIndex(index: Map<string, string>): void {
  nameIndex = index;
}

// All registered named routes as `{ name, path }` pairs (backs the agent's
// `router.getRoutes()` route-map listing).
export function getRegisteredRoutes(): { name: string; path: string }[] {
  return [...nameIndex.entries()].map(([name, path]) => ({ name, path }));
}

// Coerce a query (values may be strings, numbers, arrays, null/undefined) into
// a URL search string.
export function buildSearchString(query: NavQuery): string {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === null) continue;
    if (Array.isArray(value)) {
      for (const item of value) {
        if (item !== undefined && item !== null)
          search.append(key, String(item));
      }
    } else {
      search.set(key, String(value));
    }
  }
  return search.toString();
}

// Fill `:param` placeholders and append a query string.
export function resolvePath(
  name: string,
  options: { params?: NavParams; query?: NavQuery } = {}
): string {
  const pattern = nameIndex.get(name);
  if (!pattern) {
    // Unknown name → root, so a misconfigured redirect can't throw mid-guard.
    console.warn(`resolvePath: no route registered for name "${name}"`);
    return "/";
  }
  let path = pattern;
  if (options.params) {
    for (const [key, value] of Object.entries(options.params)) {
      if (value === undefined) continue;
      const single = Array.isArray(value) ? (value[0] ?? "") : value;
      // Match `:key` only when NOT followed by another identifier char,
      // so `:project` doesn't substring-match inside `:projectId` and
      // turn `/projects/:projectId/...` into `/projects/<v>Id/...`. The
      // collision shows up when callers merge in inherited route params
      // (e.g. the SQL editor's `:project` clobbers the issue route's
      // `:projectId`). React-router param names are restricted to
      // `[A-Za-z0-9_]+`, so the negative lookahead is safe — no regex
      // escape needed.
      path = path.replace(
        new RegExp(`:${key}(?![A-Za-z0-9_])`),
        encodeURIComponent(single)
      );
    }
  }
  if (options.query) {
    const qs = buildSearchString(options.query);
    if (qs) path = `${path}?${qs}`;
  }
  return path;
}

// The createBrowserRouter instance, registered by the app root so non-component
// code (the auth slice) can navigate without importing the root (which would
// cycle through the route lazies / app store).
let appRouter: AppRouterLike | undefined;

export function setAppRouter(router: AppRouterLike): void {
  appRouter = router;
}

export function navigateByName(
  name: string,
  options: { params?: NavParams; query?: NavQuery } & NavigationOptions = {}
): Promise<void> {
  const { params, query, ...navigationOptions } = options;
  const path = resolvePath(name, { params, query });
  return Promise.resolve(appRouter?.navigate(path, navigationOptions));
}

// Navigate to a raw path (backs `router.push(path)` / `router.replace(path)`).
export function navigateToPath(
  path: string,
  options: NavigationOptions = {}
): Promise<void> {
  return Promise.resolve(appRouter?.navigate(path, options));
}

// Current data-router state (location + matches), for the non-hook
// `router.currentRoute.value` snapshot.
export function getAppRouterState(): RouterState | undefined {
  return appRouter?.state;
}

// Subscribe to route changes (backs `router.afterEach`); returns an unregister
// fn.
export function subscribeRoute(onChange: () => void): () => void {
  if (!appRouter?.subscribe) return () => {};
  return appRouter.subscribe(() => onChange());
}

// History delta navigation (backs `router.back()` / `router.go(n)`).
export function routerGo(delta: number): void {
  appRouter?.navigate(delta);
}

// Resolves once the data router has completed its initial load (backs
// `router.isReady()`).
export function isAppRouterReady(): Promise<void> {
  if (!appRouter || appRouter.state?.initialized !== false) {
    return Promise.resolve();
  }
  return new Promise((resolve) => {
    const unsubscribe = appRouter?.subscribe?.((state) => {
      if (state.initialized) {
        unsubscribe?.();
        resolve();
      }
    });
    if (!unsubscribe) resolve();
  });
}
