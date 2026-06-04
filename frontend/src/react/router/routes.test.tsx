import type { RouteObject } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { routes } from "./routes";

// Guardrail for the "blank body" route bug class. During the Vue→React router
// migration, several leaf routes were ported as bare `{ path, handle }` objects
// with no `element`/`lazy`/`Component`. react-router renders such a leaf as an
// empty `<Outlet/>`, so the page shows nothing (e.g. `/projects/bbdev` and the
// legacy rollout/environment-detail routes). A leaf must therefore either
// render something or redirect:
//   - `lazy` / `element` / `Component` — renders a page
//   - `loader` — performs a (typically param-aware) redirect()
//   - an ancestor flagged `handle.layoutAsPage` — the ancestor renders the
//     content itself and the child Outlet is intentionally empty (SQL Editor)
//   - `handle.guardRedirect` — `rootGuard` always redirects this route (the
//     dynamic workspace-root "/" redirect that can't be a static <Navigate>)
// Any new bare leaf that doesn't fit one of these will fail this test instead
// of silently shipping a blank page.

type RouteHandle = {
  name?: string;
  layoutAsPage?: boolean;
  guardRedirect?: boolean;
};

function joinPath(parent: string, child: string): string {
  if (!child) return parent || "/";
  const left = parent.endsWith("/") ? parent.slice(0, -1) : parent;
  const right = child.startsWith("/") ? child : `/${child}`;
  return `${left}${right}` || "/";
}

function collectBareLeaves(
  list: RouteObject[],
  parentPath = "",
  underLayoutAsPage = false
): string[] {
  const bare: string[] = [];
  for (const route of list) {
    const handle = route.handle as RouteHandle | undefined;
    const segment = route.path ?? "";
    const fullPath = route.path?.startsWith("/")
      ? segment
      : joinPath(parentPath, segment);
    const layoutAsPage = underLayoutAsPage || handle?.layoutAsPage === true;
    const children = route.children;
    const isLeaf = !children || children.length === 0;
    if (isLeaf) {
      const rendersOrRedirects =
        Boolean(route.lazy) ||
        Boolean(route.element) ||
        Boolean(route.Component) ||
        Boolean(route.loader) ||
        layoutAsPage ||
        handle?.guardRedirect === true;
      if (!rendersOrRedirects) {
        bare.push(
          `${fullPath || "(index)"} [name=${handle?.name ?? "unnamed"}]`
        );
      }
    } else {
      bare.push(...collectBareLeaves(children, fullPath, layoutAsPage));
    }
  }
  return bare;
}

describe("react route table reachability", () => {
  it("every leaf route renders something or redirects (no blank-body bare leaves)", () => {
    expect(collectBareLeaves(routes)).toEqual([]);
  });
});
