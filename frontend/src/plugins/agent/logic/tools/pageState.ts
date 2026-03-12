import type { Router } from "vue-router";

export function createPageStateTool(router: Router) {
  return async (): Promise<string> => {
    const route = router.currentRoute.value;
    return JSON.stringify({
      path: route.fullPath,
      name: String(route.name ?? ""),
      params: route.params,
      query: route.query,
      title: document.title,
    });
  };
}
