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
};
type AppRouterLike = {
  // Overloaded to match the data router (`navigate(delta)` /
  // `navigate(to, opts)`), so the concrete instance is assignable.
  navigate: {
    (delta: number): Promise<void>;
    (to: string, opts?: { replace?: boolean }): Promise<void>;
  };
  subscribe?: (fn: (state: RouterState) => void) => () => void;
  state?: RouterState;
};

// vue-router navigated by route *name*; react-router navigates by *path*. To
// keep the ported guard + auth lifecycle a faithful translation (they redirect
// by name), we resolve names to concrete paths via a `name -> path pattern`
// index. The index is BUILT from the route table on the React (.tsx) side and
// REGISTERED here (`setRouteNameIndex`) — this module stays a pure `.ts` helper.

// vue-router accepted strings, numbers and arrays as query/param values.
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

// Coerce a vue-router-style query (values may be strings, numbers, arrays,
// null/undefined) into a URL search string.
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

// The createBrowserRouter instance, registered by the app root in Phase 4 so
// non-component code (the auth slice) can navigate without importing the root
// (which would cycle through the route lazies / app store).
let appRouter: AppRouterLike | undefined;

export function setAppRouter(router: AppRouterLike): void {
  appRouter = router;
}

// Navigate by route name (mirrors vue-router `router.push({ name, query })`).
export function navigateByName(
  name: string,
  options: {
    params?: NavParams;
    query?: NavQuery;
    replace?: boolean;
  } = {}
): Promise<void> {
  const path = resolvePath(name, options);
  return Promise.resolve(
    appRouter?.navigate(path, { replace: options.replace })
  );
}

// Navigate to a raw path (mirrors `router.push(path)` / `router.replace(path)`).
export function navigateToPath(
  path: string,
  options: { replace?: boolean } = {}
): Promise<void> {
  return Promise.resolve(
    appRouter?.navigate(path, { replace: options.replace })
  );
}

// Current data-router state (location + matches), for non-hook snapshots used
// by the `router.currentRoute.value` drop-in.
export function getAppRouterState(): RouterState | undefined {
  return appRouter?.state;
}

// Subscribe to route changes (mirrors vue-router `router.afterEach`); returns
// an unregister fn.
export function subscribeRoute(onChange: () => void): () => void {
  if (!appRouter?.subscribe) return () => {};
  return appRouter.subscribe(() => onChange());
}

// History delta navigation (mirrors `router.back()` / `router.go(n)`).
export function routerGo(delta: number): void {
  appRouter?.navigate(delta);
}

// Resolves once the data router has completed its initial load (mirrors
// vue-router `router.isReady()`).
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
