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

// react-router runs `rootLoader` during initial hydration. With partial
// hydration (the v7 default) and no hydrate fallback, react-router first
// renders the matched route tree — for "/" that is the element-less
// `WORKSPACE_ROOT_MODULE` index route — for one tick before the loader's
// `redirect()` resolves, logging "No HydrateFallback element provided" and
// "Matched leaf route at location '/' does not have an element". The loader
// resolves synchronously (it reads the app store and returns a redirect), so a
// fallback that renders nothing is shown for ~0 frames; defining one satisfies
// react-router's partial-hydration contract and silences both warnings.
routes[0].HydrateFallback = () => null;

export const appRouter = createBrowserRouter(routes);

// Let non-component code (the auth slice) navigate through this instance.
setAppRouter(appRouter);

export function AppRoot() {
  return <RouterProvider router={appRouter} />;
}
