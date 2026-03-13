import type { Router } from "vue-router";
import { lazyExtractDomTree } from "../../dom";
import { extractRouteContext } from "../context";

export interface PageStateArgs {
  mode?: "semantic" | "dom";
}

export function createPageStateTool(router: Router) {
  return async (args?: PageStateArgs): Promise<string> => {
    const route = router.currentRoute.value;
    const base: Record<string, unknown> = {
      path: route.fullPath,
      name: String(route.name ?? ""),
      params: route.params,
      query: route.query,
      title: document.title,
    };

    // Enrich with Pinia store data
    const ctx = await extractRouteContext(route);
    if (Object.keys(ctx).length > 0) {
      base.context = ctx;
    }

    if (args?.mode === "dom") {
      const { tree, count } = await lazyExtractDomTree();
      return JSON.stringify({
        ...base,
        interactiveElements: count,
        domTree: tree,
      });
    }

    return JSON.stringify(base);
  };
}
