import type { Router } from "vue-router";
import type { DomActionParams, DomActionResult } from "./actions";

// Single dynamic import path. actions.ts re-exports extractDomTree and
// getElementByIndex from domTree.ts, ensuring one module instance so the
// element registry populated by extractDomTree is readable by getElementByIndex.
let module: typeof import("./actions") | undefined;

async function ensureLoaded(): Promise<typeof import("./actions")> {
  if (module) return module;
  module = await import("./actions");
  return module;
}

export async function lazyExtractDomTree(): Promise<{
  tree: string;
  count: number;
}> {
  const m = await ensureLoaded();
  return m.extractDomTree();
}

function isSameOriginLink(el: Element): string | undefined {
  if (!(el instanceof HTMLAnchorElement)) return undefined;
  const href = el.getAttribute("href");
  if (!href) return undefined;
  // Relative paths or same-origin absolute URLs
  if (href.startsWith("/")) return href;
  try {
    const url = new URL(href);
    if (url.origin === window.location.origin)
      return url.pathname + url.search + url.hash;
  } catch {
    // Not a valid URL
  }
  return undefined;
}

export async function lazyExecuteDomAction(
  params: DomActionParams,
  router?: Router
): Promise<DomActionResult> {
  const m = await ensureLoaded();
  const entry = m.getElementByIndex(params.index);
  if (!entry) {
    return {
      success: false,
      message: `Element [${params.index}] not found. Run get_page_state(mode="dom") to refresh the DOM tree.`,
    };
  }

  // For <a> clicks, use Vue Router to avoid full page reload
  if (params.type === "click" && router) {
    const path = isSameOriginLink(entry.element);
    if (path) {
      await router.push(path);
      return {
        success: true,
        message: `Navigated to ${path}`,
      };
    }
  }

  return m.executeDomAction(params, entry.element);
}

export type { DomActionParams, DomActionResult };
