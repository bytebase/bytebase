import { useCallback } from "react";
import type {
  DashboardShellTargets,
  ReactRouteShellTargets,
} from "@/app/dashboard-shell";
import { DashboardBodyShell } from "@/components/DashboardBodyShell";

interface IssuesRouteShellProps {
  routeKey?: string;
  onReady?: (targets: ReactRouteShellTargets | null) => void;
}

export function IssuesRouteShell({ routeKey, onReady }: IssuesRouteShellProps) {
  const handleReady = useCallback(
    (targets: DashboardShellTargets) => {
      onReady?.({
        content: targets.content,
        quickstart: targets.quickstart,
        mainContainer: targets.mainContainer,
      });
    },
    [onReady]
  );

  return (
    <DashboardBodyShell
      variant="issues"
      routeKey={routeKey}
      onReady={handleReady}
    />
  );
}
