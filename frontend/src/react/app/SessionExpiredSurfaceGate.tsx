import { SessionExpiredSurface } from "@/react/components/auth/SessionExpiredSurface";
import { useCurrentRoute } from "@/react/router";
import { useAppStore } from "@/react/stores/app";
import { isAuthRelatedRoute } from "@/utils/auth";

export function SessionExpiredSurfaceGate() {
  const route = useCurrentRoute();
  const isAuthRoute = isAuthRelatedRoute(String(route.name ?? ""));
  const isLoggedIn = useAppStore((s) => s.isLoggedIn());
  const unauthenticatedOccurred = useAppStore((s) => s.unauthenticatedOccurred);
  const currentPath = route.fullPath;
  // Match the guards that previously lived in AuthContext.vue: the
  // surface is only shown for an already-signed-in user on a non-auth
  // route who just lost their session — otherwise the modal would block
  // signin/signup flows.
  if (isAuthRoute || !isLoggedIn || !unauthenticatedOccurred) return null;
  return <SessionExpiredSurface currentPath={currentPath} />;
}
