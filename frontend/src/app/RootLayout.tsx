import { useRef } from "react";
import type { Location } from "react-router";
import { matchRoutes, Outlet, useBlocker } from "react-router";
import { AuthGate } from "@/app/AuthGate";
import { RouteBehaviorRecorder } from "@/app/analytics/RouteBehaviorRecorder";
import type { ReactRoute } from "@/app/router";
import { buildReactRoute, runBeforeEachGuards } from "@/app/router";
import { NavigationScrollRestoration } from "@/app/router/NavigationScrollRestoration";
import { routes } from "@/app/router/routes";
import { SessionExpiredSurfaceGate } from "@/app/SessionExpiredSurfaceGate";
import { Toaster } from "@/components/ui/toaster";
import { Watermark } from "@/components/Watermark";
import { AgentWindow } from "@/modules/agent/components/AgentWindow";

// Translate a react-router Location into the legacy `ReactRoute` snapshot the
// registered `beforeEach` leave guards expect (they read `to.name` /
// `to.fullPath`). Route name comes from matching the path against the table.
function snapshotLocation(location: Location): ReactRoute {
  const matched = matchRoutes(routes, location.pathname) ?? [];
  const leaf = matched.at(-1);
  return buildReactRoute(
    {
      pathname: location.pathname,
      search: location.search,
      hash: location.hash,
    },
    matched.map((m) => ({ handle: m.route.handle })),
    (leaf?.params ?? {}) as Record<string, string | string[] | undefined>
  );
}

// Single blocker reproducing vue-router's global `beforeEach` cancellation:
// consults every registered leave guard and blocks the navigation if any calls
// `next(false)`. The guards run `window.confirm` synchronously and remember a
// pending target themselves, so a blocked navigation simply stays put.
function LeaveGuardBlocker() {
  const blockerRef = useRef<ReturnType<typeof useBlocker> | null>(null);
  const blocker = useBlocker(
    ({ currentLocation, historyAction, nextLocation }) => {
      if (
        currentLocation.pathname === nextLocation.pathname &&
        currentLocation.search === nextLocation.search
      ) {
        return false;
      }
      return runBeforeEachGuards(
        snapshotLocation(nextLocation),
        snapshotLocation(currentLocation),
        {
          historyAction,
          reset: () => blockerRef.current?.reset?.(),
          retry: () => blockerRef.current?.proceed?.(),
        }
      );
    }
  );
  blockerRef.current = blocker;
  return null;
}

// Root route element for the react-router app shell. Hosts the global overlays
// that previously lived in `ReactApp.tsx` (Watermark / Toaster / AgentWindow /
// SessionExpiredSurfaceGate) and wraps the routed tree in `<AuthGate>` so the
// session lifecycle (load gate, poll, cross-tab switch, inactivity reminder)
// runs around every page.
export function RootLayout() {
  return (
    <>
      <Watermark />
      <Toaster />
      <AgentWindow />
      <SessionExpiredSurfaceGate />
      <LeaveGuardBlocker />
      <RouteBehaviorRecorder />
      <NavigationScrollRestoration>
        <AuthGate>
          <Outlet />
        </AuthGate>
      </NavigationScrollRestoration>
    </>
  );
}
