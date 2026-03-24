import type { Router } from "vue-router";
import type { DomActionParams, DomActionResult } from "./actions";

// Single dynamic import path. actions.ts re-exports extractDomTree and
// getElementByRef from domTree.ts, ensuring one module instance so the
// element registry populated by extractDomTree is readable by getElementByRef.
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
    if (url.origin === window.location.origin) {
      return url.pathname + url.search + url.hash;
    }
  } catch {
    // Not a valid URL
  }
  return undefined;
}

function formatRefreshError(message: string): DomActionResult {
  return {
    success: false,
    message: `${message} Run get_page_state(mode="dom") to refresh the DOM tree.`,
  };
}

function getElementRef(params: DomActionParams): string | undefined {
  return typeof params.index === "string" ? params.index.trim() : undefined;
}

export async function lazyExecuteDomAction(
  params: DomActionParams,
  router?: Router
): Promise<DomActionResult> {
  const elementRef = getElementRef(params);
  if (!elementRef) {
    return formatRefreshError(
      `Invalid element ref: ${JSON.stringify(params.index)}. Use refs like [e1] from the DOM tree.`
    );
  }
  if (!/^e\d+$/.test(elementRef)) {
    return formatRefreshError(
      `Malformed element ref [${elementRef}]. Use refs like [e1] from the DOM tree.`
    );
  }

  const m = await ensureLoaded();
  const entry = m.getElementByRef(elementRef);
  if (!entry) {
    return formatRefreshError(`Element [${elementRef}] was not found.`);
  }
  if (!entry.element.isConnected) {
    return formatRefreshError(`Element [${elementRef}] is stale.`);
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
