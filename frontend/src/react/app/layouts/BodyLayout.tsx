import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { Outlet, useLocation, useMatches } from "react-router-dom";
import { DashboardBodyShell } from "@/react/components/DashboardBodyShell";
import { DashboardSidebar } from "@/react/components/DashboardSidebar";
import { ProjectSidebar } from "@/react/components/ProjectSidebar";
import { Quickstart } from "@/react/components/Quickstart";
import { RoutePermissionGuardShell } from "@/react/components/RoutePermissionGuardShell";
import type { DashboardShellTargets } from "@/react/dashboard-shell";
import {
  PROJECT_V1_ROUTE_DASHBOARD,
  WORKSPACE_ROOT_MODULE,
  WORKSPACE_ROUTE_MY_ISSUES,
} from "@/react/router/handles";

// Ported from `src/layouts/BodyLayout.vue`. Mounts the workspace
// `DashboardBodyShell` (header + sidebar slot + body slot) and portals the
// React sidebar, routed body (`<Outlet/>` — the Vue named views collapse to
// one outlet here, gated by `RoutePermissionGuardShell` on non-project
// routes), and quickstart into the shell's reported targets. The
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
  // Workspace "My Issues" is a standalone full-width page: header (with logo)
  // but no sidebar, mirroring the Vue IssuesRouteShell (variant="issues").
  const isMyIssues = currentRouteName === WORKSPACE_ROUTE_MY_ISSUES;
  // Project-scoped routes (`workspace.project.*`) get the project sidebar; the
  // bare projects list (`workspace.project`) keeps the workspace sidebar.
  const isProjectRoute = Boolean(
    currentRouteName?.startsWith(`${PROJECT_V1_ROUTE_DASHBOARD}.`)
  );
  const Sidebar = isProjectRoute ? ProjectSidebar : DashboardSidebar;
  const variant = isMyIssues ? "issues" : "workspace";
  const routeKey = `${location.pathname}${location.search}`;

  const [targets, setTargets] = useState<DashboardShellTargets>({
    desktopSidebar: null,
    mobileSidebar: null,
    content: null,
    quickstart: null,
    mainContainer: null,
  });

  // Non-project routes render through `RoutePermissionGuardShell`: it mounts
  // into the shell's content target and reports a non-null target only once the
  // route's `requiredPermissions` are satisfied, so a restricted page never
  // mounts (it shows the permission-denied / request-role fallback instead).
  // This mirrors the Vue `BodyLayout.vue` content teleport. Project routes
  // (`workspace.project.*`) bypass this gate — their gating belongs to
  // `ProjectRouteShell`, which loads the project resource for project-scoped
  // permission checks.
  const [permissionTarget, setPermissionTarget] =
    useState<HTMLDivElement | null>(null);

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
        variant={variant}
        isRootPath={isRootPath}
        routeKey={routeKey}
        onReady={setTargets}
      />

      {showSidebar
        ? sidebarTargets.map((target, i) =>
            target ? createPortal(<Sidebar />, target, `sidebar-${i}`) : null
          )
        : null}

      {isProjectRoute
        ? targets.content
          ? createPortal(<Outlet />, targets.content)
          : null
        : targets.content
          ? createPortal(
              <RoutePermissionGuardShell
                routeKey={routeKey}
                className="m-4"
                targetClassName="h-full min-h-0"
                onReady={setPermissionTarget}
              />,
              targets.content
            )
          : null}

      {!isProjectRoute && permissionTarget
        ? createPortal(<Outlet />, permissionTarget)
        : null}

      {targets.quickstart
        ? createPortal(<Quickstart />, targets.quickstart)
        : null}
    </>
  );
}
