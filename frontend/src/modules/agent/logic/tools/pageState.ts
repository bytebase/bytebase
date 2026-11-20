import type { AppRouterInstance } from "@/app/router";
import { lazyExtractDomTree } from "../../dom";
import {
  type AgentDomContext,
  isDomContextCurrent,
} from "../../dom/interactionRegistry";
import { extractRouteContext } from "../context";

export interface PageStateArgs {
  mode?: "semantic" | "dom";
}

export function createPageStateTool(
  router: AppRouterInstance,
  getContext: () => AgentDomContext | undefined
) {
  return async (args?: PageStateArgs): Promise<string> => {
    const route = router.currentRoute.value;
    const context = getContext();
    const current = () =>
      context &&
      isDomContextCurrent(context) &&
      router.currentRoute.value.fullPath === context.pageKey;
    const stale = JSON.stringify({
      error:
        "The selected page or run changed. Inspect the current page again.",
    });
    if (!current()) return stale;
    const base: Record<string, unknown> = {
      path: route.fullPath,
      name: String(route.name ?? ""),
      params: route.params,
      query: route.query,
      title: document.title,
    };

    // Enrich with app store data
    const ctx = await extractRouteContext(route);
    if (!current()) return stale;
    if (Object.keys(ctx).length > 0) {
      base.context = ctx;
    }

    if (args?.mode === "dom") {
      const { tree, count } = await lazyExtractDomTree(context);
      if (!current()) return stale;
      return JSON.stringify({
        ...base,
        interactiveElements: count,
        domTree: tree,
      });
    }

    return JSON.stringify(base);
  };
}
