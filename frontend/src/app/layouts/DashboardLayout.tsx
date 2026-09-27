import { useState } from "react";
import { createPortal } from "react-dom";
import { Outlet } from "react-router";
import type { DashboardFrameShellTargets } from "@/app/dashboard-shell";
import { DashboardFrameShell } from "@/components/DashboardFrameShell";
import { useEnsureWorkspaceCommonData } from "@/hooks/useEnsureWorkspaceCommonData";

// Mounts `DashboardFrameShell` and portals the routed `<Outlet/>` into the body
// element the shell reports via `onReady`. The shell renders banners + a
// loading gate; this layout also kicks off the workspace-scope bootstrap
// (idempotent, deduped by the app store).
export function DashboardLayout() {
  useEnsureWorkspaceCommonData();

  const [targets, setTargets] = useState<DashboardFrameShellTargets>({
    banner: null,
    body: null,
  });

  return (
    <>
      <DashboardFrameShell onReady={setTargets} />
      {targets.body ? createPortal(<Outlet />, targets.body) : null}
    </>
  );
}
