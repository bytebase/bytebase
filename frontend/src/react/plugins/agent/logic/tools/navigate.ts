import type { Router } from "vue-router";

function getRouteMap(router: Router): string[] {
  const routes = router.getRoutes();
  const paths = new Set<string>();

  for (const route of routes) {
    const path = route.path;
    // Skip empty, auth-only, and error routes
    if (
      !path ||
      path.startsWith("/auth") ||
      path === "/setup" ||
      path === "/403" ||
      path === "/404"
    ) {
      continue;
    }
    // Skip internal layout-only routes (no name, have children)
    if (!route.name && route.children.length > 0) continue;

    paths.add(path);
  }

  return [...paths].sort();
}

export function createNavigateTool(router: Router) {
  return async (args: { path?: string; list?: boolean }): Promise<string> => {
    if (args.list) {
      const routes = getRouteMap(router);
      return JSON.stringify({ routes });
    }

    if (!args.path) {
      return JSON.stringify({ error: "path is required when list is not set" });
    }

    try {
      await router.push(args.path);
      return JSON.stringify({
        navigated: true,
        currentPath: router.currentRoute.value.fullPath,
      });
    } catch (err) {
      return JSON.stringify({
        error: `Navigation failed: ${err instanceof Error ? err.message : String(err)}`,
      });
    }
  };
}
