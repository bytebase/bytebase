import { SessionExpiredSurface } from "@/react/components/auth/SessionExpiredSurface";
import { useVueState } from "@/react/hooks/useVueState";
import { useCurrentRoute } from "@/react/router";
import { useAuthStore } from "@/store";
import { isAuthRelatedRoute } from "@/utils/auth";

export function SessionExpiredSurfaceGate() {
  const route = useCurrentRoute();
  const isAuthRoute = isAuthRelatedRoute(String(route.name ?? ""));
  const isLoggedIn = useVueState(() => useAuthStore().isLoggedIn);
  const unauthenticatedOccurred = useVueState(
    () => useAuthStore().unauthenticatedOccurred
  );
  const currentPath = route.fullPath;
  // Match the guards that previously lived in AuthContext.vue: the
  // surface is only shown for an already-signed-in user on a non-auth
  // route who just lost their session — otherwise the modal would block
  // signin/signup flows.
  if (isAuthRoute || !isLoggedIn || !unauthenticatedOccurred) return null;
  return <SessionExpiredSurface currentPath={currentPath} />;
}
