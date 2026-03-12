import type { Router } from "vue-router";

export function createNavigateTool(router: Router) {
  return async (args: { path: string }): Promise<string> => {
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
