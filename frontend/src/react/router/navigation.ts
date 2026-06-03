// Structural type for the createBrowserRouter instance (avoids depending on
// @remix-run/router's exported name across react-router versions).
type AppRouterLike = {
  navigate: (to: string, opts?: { replace?: boolean }) => unknown;
};

// vue-router navigated by route *name*; react-router navigates by *path*. To
// keep the ported guard + auth lifecycle a faithful translation (they redirect
// by name), we resolve names to concrete paths via a `name -> path pattern`
// index. The index is BUILT from the route table on the React (.tsx) side and
// REGISTERED here (`setRouteNameIndex`) — this module stays a pure `.ts` so the
// app store (also `.ts`, checked by vue-tsc) can import it without pulling the
// `.tsx` route/page graph across the type-check project boundary.

type NavQuery = Record<string, string | undefined>;

let nameIndex = new Map<string, string>();

export function setRouteNameIndex(index: Map<string, string>): void {
  nameIndex = index;
}

// Fill `:param` placeholders and append a query string.
export function resolvePath(
  name: string,
  options: { params?: Record<string, string>; query?: NavQuery } = {}
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
      path = path.replace(`:${key}`, encodeURIComponent(value));
    }
  }
  const query = options.query;
  if (query) {
    const search = new URLSearchParams();
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined) search.set(key, value);
    }
    const qs = search.toString();
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
    params?: Record<string, string>;
    query?: NavQuery;
    replace?: boolean;
  } = {}
): void {
  const path = resolvePath(name, options);
  appRouter?.navigate(path, { replace: options.replace });
}

// Navigate to a raw path (mirrors `router.push(path)` / `router.replace(path)`).
export function navigateToPath(
  path: string,
  options: { replace?: boolean } = {}
): void {
  appRouter?.navigate(path, { replace: options.replace });
}
