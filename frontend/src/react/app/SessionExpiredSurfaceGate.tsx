import { SessionExpiredSurface } from "@/react/components/auth/SessionExpiredSurface";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { useAuthStore } from "@/store";

export function SessionExpiredSurfaceGate() {
  const unauthenticatedOccurred = useVueState(
    () => useAuthStore().unauthenticatedOccurred
  );
  const currentPath = useVueState(() => router.currentRoute.value.fullPath);
  if (!unauthenticatedOccurred) return null;
  return <SessionExpiredSurface currentPath={currentPath} />;
}
