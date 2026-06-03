import { useAppStore } from "@/react/stores/app";

/**
 * Resolve the active workspace resource name for auth flows (signin, MFA,
 * password reset/forgot, OAuth callback). Prefers the app store's resolved
 * workspace (from server info); falls back to the `?workspace=<id>` URL query.
 *
 * Lives in the React layer (not `@/utils`) because it reads the Zustand app
 * store — keeping the app-store import out of the low-level `@/utils` barrel,
 * which app-store slices themselves import (avoids a load-time cycle).
 */
export const resolveWorkspaceName = (): string | undefined => {
  const workspaceID =
    new URLSearchParams(window.location.search).get("workspace") ?? undefined;
  const workspaceNameFromQuery = workspaceID
    ? `workspaces/${workspaceID}`
    : undefined;
  return (
    useAppStore.getState().workspaceResourceName() || workspaceNameFromQuery
  );
};
