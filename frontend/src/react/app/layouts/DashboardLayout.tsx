import { useState } from "react";
import { createPortal } from "react-dom";
import { Outlet } from "react-router-dom";
import { DashboardFrameShell } from "@/react/components/DashboardFrameShell";
import type { DashboardFrameShellTargets } from "@/react/dashboard-shell";
import { useEnsureWorkspaceCommonData } from "@/react/hooks/useEnsureWorkspaceCommonData";

// Ported from `src/layouts/DashboardLayout.vue`. The Vue layout mounted
// `DashboardFrameShell` and teleported `<router-view name="body"/>` into the
// shell's body target. In React the named view collapses to the single
// `<Outlet/>`, portaled into the body element the shell reports via `onReady`.
// The shell renders banners + a loading gate; this layout also kicks off the
// workspace-scope bootstrap (idempotent, deduped by the app store).
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
