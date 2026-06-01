import { useEffect, useState } from "react";
import { useAppStore } from "@/react/stores/app";

/**
 * Workspace-scope common data bootstrap.
 *
 * Every top-level React shell (DashboardFrameShell, SQLEditorLayout, and any
 * future scope shell) calls this once. All loaders are idempotent (they
 * dedupe via in-flight request refs in the app store), so wiring this hook
 * up at multiple shells does not produce duplicate gRPC calls — it just
 * guarantees the data is loading by the time any descendant page mounts.
 *
 * Pages should not call these loaders individually any more. The whole
 * point of this hook is to remove the "did I remember to fetch on this
 * page?" failure mode. Anything that needs to wait until permissions are
 * ready can use `usePermissionDataReady()` from ComponentPermissionGuard,
 * which awaits the same idempotent request.
 *
 * Auth-flow pages (SetupPage, ProfileSetupPage, OAuth2ConsentPage) render
 * outside any shell; they still handle their own loads because they need
 * a subset of this data before any shell would otherwise mount.
 *
 * Returns `true` once the initial bootstrap promise has settled. Shells
 * use it to gate their loading spinner.
 */
export function useEnsureWorkspaceCommonData(): boolean {
  const loadCurrentUser = useAppStore((state) => state.loadCurrentUser);
  const loadServerInfo = useAppStore((state) => state.loadServerInfo);
  const loadWorkspace = useAppStore((state) => state.loadWorkspace);
  const loadWorkspaceProfile = useAppStore(
    (state) => state.loadWorkspaceProfile
  );
  const loadEnvironmentList = useAppStore((state) => state.loadEnvironmentList);
  const loadWorkspacePermissionState = useAppStore(
    (state) => state.loadWorkspacePermissionState
  );
  const loadSubscription = useAppStore((state) => state.loadSubscription);

  const [ready, setReady] = useState(false);

  useEffect(() => {
    let cancelled = false;
    void Promise.all([
      loadCurrentUser(),
      loadServerInfo(),
      loadWorkspace(),
      loadWorkspaceProfile(),
      loadEnvironmentList(),
      loadWorkspacePermissionState(),
      loadSubscription(),
    ])
      .catch(() => undefined)
      .then(() => {
        if (!cancelled) {
          setReady(true);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [
    loadCurrentUser,
    loadServerInfo,
    loadWorkspace,
    loadWorkspaceProfile,
    loadEnvironmentList,
    loadWorkspacePermissionState,
    loadSubscription,
  ]);

  return ready;
}
