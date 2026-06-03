import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { Outlet, useLocation, useMatches } from "react-router-dom";
import { DashboardBodyShell } from "@/react/components/DashboardBodyShell";
import { DashboardSidebar } from "@/react/components/DashboardSidebar";
import { Quickstart } from "@/react/components/Quickstart";
import type { DashboardShellTargets } from "@/react/dashboard-shell";
import { WORKSPACE_ROOT_MODULE } from "@/react/router/handles";

// Ported from `src/layouts/BodyLayout.vue`. Mounts the workspace
// `DashboardBodyShell` (header + sidebar slot + body slot) and portals the
// React sidebar, routed body (`<Outlet/>` — the Vue named views collapse to
// one outlet here), and quickstart into the shell's reported targets. The
// shell owns the responsive sidebar / mobile drawer; this layout supplies the
// React `DashboardSidebar` tree directly instead of the Vue ReactSidebarMount.
// Also ports the agent keyboard shortcut from the Vue layout's lifecycle.
//
// NOTE: the Vue layout's periodic "refresh reminder" (actuatorStore
// .tryToRemindRefresh) is intentionally omitted — it reads Pinia-only actuator
// state with no app-store equivalent yet, and this layout is restricted to
// `useAppStore` / react-router reads. Reattach once that surface is ported.
export function BodyLayout() {
  const matches = useMatches();
  const location = useLocation();
  const currentRouteName = (
    matches.at(-1)?.handle as { name?: string } | undefined
  )?.name;
  const isRootPath = currentRouteName === WORKSPACE_ROOT_MODULE;
  const routeKey = `${location.pathname}${location.search}`;

  const [targets, setTargets] = useState<DashboardShellTargets>({
    desktopSidebar: null,
    mobileSidebar: null,
    content: null,
    quickstart: null,
    mainContainer: null,
  });

  // Agent toggle shortcut (Ctrl/Cmd+Shift+A), ported from the Vue layout.
  useEffect(() => {
    const handler = async (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === "A") {
        e.preventDefault();
        const { useAgentStore } = await import(
          "@/react/plugins/agent/store/agent"
        );
        useAgentStore.getState().toggle();
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  const showSidebar = !isRootPath;
  const sidebarTargets = [targets.desktopSidebar, targets.mobileSidebar];

  return (
    <>
      <DashboardBodyShell
        variant="workspace"
        isRootPath={isRootPath}
        routeKey={routeKey}
        onReady={setTargets}
      />

      {showSidebar
        ? sidebarTargets.map((target, i) =>
            target
              ? createPortal(<DashboardSidebar />, target, `sidebar-${i}`)
              : null
          )
        : null}

      {targets.content ? createPortal(<Outlet />, targets.content) : null}

      {targets.quickstart
        ? createPortal(<Quickstart />, targets.quickstart)
        : null}
    </>
  );
}
