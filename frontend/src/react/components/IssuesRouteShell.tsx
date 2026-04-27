import { useCallback } from "react";
import { DashboardBodyShell } from "@/react/components/DashboardBodyShell";
import type {
  DashboardShellTargets,
  ReactRouteShellTargets,
} from "@/react/dashboard-shell";

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
