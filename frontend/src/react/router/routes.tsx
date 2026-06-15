import type { RouteObject } from "react-router-dom";
import { RootLayout } from "@/react/app/RootLayout";
import { RouteErrorPage } from "@/react/app/RouteErrorPage";
import { rootGuard } from "@/react/router/guard";
import { WORKSPACE_ROUTE_404 } from "@/react/router/handles";
import { authRoutes } from "@/react/router/routes/auth";
import { dashboardRoutes } from "@/react/router/routes/dashboard";
import { sqlEditorRoutes } from "@/react/router/routes/sqlEditor";

// React-router route table translated from the vue-router route definitions
// (`@/router/auth.ts`, `@/router/setup.ts`, `@/router/sqlEditor.ts`,
// `@/router/dashboard/**`). Composed in the same order as the vue router's
// `routes: [...authRoutes, ...setupRoutes, ...dashboardRoutes,
// ...sqlEditorRoutes]` (setup is folded into authRoutes here, both rendering
// the SplashLayout). The whole table hangs off a single `RootLayout` route that
// hosts the global overlays + `<AuthGate>`. Wiring this table into the
// application root happens in a later phase.
export const routes: RouteObject[] = [
  {
    element: <RootLayout />,
    // Catch-all for uncaught render/loader exceptions anywhere in the
    // tree: show a user-facing recovery page instead of react-router's
    // developer default screen.
    errorElement: <RouteErrorPage />,
    children: [
      ...authRoutes,
      ...dashboardRoutes,
      ...sqlEditorRoutes,
      {
        path: "*",
        handle: { name: WORKSPACE_ROUTE_404 },
        loader: ({ request }) =>
          rootGuard({
            name: WORKSPACE_ROUTE_404,
            url: new URL(request.url),
          }),
      },
    ],
  },
];

function joinPath(parent: string, child: string): string {
  if (!child) return parent || "/";
  const left = parent.endsWith("/") ? parent.slice(0, -1) : parent;
  const right = child.startsWith("/") ? child : `/${child}`;
  return `${left}${right}` || "/";
}

// Flatten the nested route table into a `name -> full path pattern` map,
// joining parent/child segments. Registered into `navigation.ts` at app boot so
// the ported guard + auth lifecycle can resolve their by-name redirects to
// paths. Lives here (a `.tsx` module) so `navigation.ts` stays a pure `.ts`.
export function buildRouteNameIndex(
  list: RouteObject[] = routes,
  parentPath = ""
): Map<string, string> {
  const index = new Map<string, string>();
  for (const route of list) {
    const segment = route.path ?? "";
    const fullPath = route.path?.startsWith("/")
      ? segment
      : joinPath(parentPath, segment);
    const name = (route.handle as { name?: string } | undefined)?.name;
    if (name && !index.has(name)) {
      index.set(name, fullPath || "/");
    }
    if (route.children) {
      for (const [childName, childPath] of buildRouteNameIndex(
        route.children,
        fullPath
      )) {
        if (!index.has(childName)) index.set(childName, childPath);
      }
    }
  }
  return index;
}
