import { useRef } from "react";
import type { Location } from "react-router-dom";
import { matchRoutes, Outlet, useBlocker } from "react-router-dom";
import { AuthGate } from "@/react/app/AuthGate";
import { SessionExpiredSurfaceGate } from "@/react/app/SessionExpiredSurfaceGate";
import { Toaster } from "@/react/components/ui/toaster";
import { Watermark } from "@/react/components/Watermark";
import { AgentWindow } from "@/react/plugins/agent/components/AgentWindow";
import type { ReactRoute } from "@/react/router";
import { buildReactRoute, runBeforeEachGuards } from "@/react/router";
import { routes } from "@/react/router/routes";

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
      <AuthGate>
        <Outlet />
      </AuthGate>
    </>
  );
}
