import { type ComponentType, createElement, useMemo } from "react";
import { useLocation, useMatches, useParams } from "react-router-dom";

// Adapter for react-router's `lazy` route field.
//
// Page components were authored for the Vue layer, which injected route data as
// props. react-router renders a route's `Component` with no props, so we wrap
// the resolved page in a thin component that forwards the current route data:
//   - `useParams()` → the `:param` props (projectId, instanceId, databaseName,
//     issueId, …). Without these the project issue dashboard built a
//     `project:<projectId>` scope from `undefined` and crashed.
//   - `routeName` → the matched leaf route's `handle.name`, and `routeQuery` →
//     the parsed query string. Pages like the plan detail decide which phase /
//     stage / task / spec to show from these (mirroring the Vue `route.name` /
//     `route.query`); without injection that selection silently never fires.
// Pages that don't declare these props simply ignore the extras.
export const lazyPage =
  <T extends Record<string, unknown>>(
    loader: () => Promise<T>,
    pick: (m: T) => unknown
  ) =>
  async (): Promise<{ Component: ComponentType }> => {
    const m = await loader();
    const Page = pick(m) as ComponentType<Record<string, unknown>>;
    const WithRouteProps = () => {
      const params = useParams();
      const matches = useMatches();
      const { search } = useLocation();
      const routeName = (
        matches.at(-1)?.handle as { name?: string } | undefined
      )?.name;
      // Memoize on the raw search string so the query object keeps a stable
      // identity across re-renders that don't change the query (a fresh object
      // every render would needlessly re-run query-driven effects downstream).
      const routeQuery = useMemo(
        () => Object.fromEntries(new URLSearchParams(search)),
        [search]
      );
      return createElement(Page, { ...params, routeName, routeQuery });
    };
    return { Component: WithRouteProps };
  };
