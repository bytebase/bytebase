import {
  createBrowserRouter,
  type LoaderFunctionArgs,
  matchRoutes,
  RouterProvider,
} from "react-router-dom";
import { rootGuard } from "@/react/router/guard";
import { setAppRouter, setRouteNameIndex } from "@/react/router/navigation";
import { buildRouteNameIndex, routes } from "@/react/router/routes";

// Register the name→path index so the guard + auth lifecycle can resolve their
// by-name redirects to paths.
setRouteNameIndex(buildRouteNameIndex());

// Single root-route loader: runs on every navigation (the root route matches
// every URL). Resolves the matched leaf route's `handle.name` and delegates to
// the faithful `beforeEach` port; a returned `redirect()` Response navigates.
function rootLoader({ request }: LoaderFunctionArgs) {
  const url = new URL(request.url);
  const matched = matchRoutes(routes, url.pathname);
  const name = (matched?.at(-1)?.route.handle as { name?: string } | undefined)
    ?.name;
  return rootGuard({ name, url });
}

// Attach the guard to the root (RootLayout) route.
routes[0].loader = rootLoader;

export const appRouter = createBrowserRouter(routes);

// Let non-component code (the auth slice) navigate through this instance.
setAppRouter(appRouter);

export function AppRoot() {
  return <RouterProvider router={appRouter} />;
}
