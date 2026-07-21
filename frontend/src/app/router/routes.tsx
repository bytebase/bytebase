import type { RouteObject } from "react-router";
import { RootLayout } from "@/app/RootLayout";
import { RouteErrorPage } from "@/app/RouteErrorPage";
import { rootGuard } from "@/app/router/guard";
import { WORKSPACE_ROUTE_404 } from "@/app/router/handles";
import { authRoutes } from "@/app/router/routes/auth";
import { dashboardRoutes } from "@/app/router/routes/dashboard";
import { sqlEditorRoutes } from "@/app/router/routes/sqlEditor";

// Compose route groups under the root providers. Keep the catch-all last so
// React Router can rank all concrete application paths first.
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
