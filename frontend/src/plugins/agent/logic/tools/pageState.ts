import type { Router } from "vue-router";
import { lazyExtractDomTree } from "../../dom";

export interface PageStateArgs {
  mode?: "semantic" | "dom";
}

export function createPageStateTool(router: Router) {
  return async (args?: PageStateArgs): Promise<string> => {
    const route = router.currentRoute.value;
    const base = {
      path: route.fullPath,
      name: String(route.name ?? ""),
      params: route.params,
      query: route.query,
      title: document.title,
    };

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
