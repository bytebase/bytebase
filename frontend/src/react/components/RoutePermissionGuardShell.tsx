import { useEffect, useRef } from "react";
import {
  ComponentPermissionGuard,
  useComponentPermissionState,
} from "@/react/components/ComponentPermissionGuard";
import { useCurrentRoute } from "@/react/router";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

export interface RoutePermissionGuardShellProps {
  project?: Project;
  className?: string;
  targetClassName?: string;
  routeKey?: string;
  onReady?: (target: HTMLDivElement | null) => void;
}

export function RoutePermissionGuardShell({
  project,
  className,
  targetClassName,
  routeKey,
  onReady,
}: RoutePermissionGuardShellProps) {
  const route = useCurrentRoute();
  const targetRef = useRef<HTMLDivElement>(null);
  const permissions = route.requiredPermissions;
  const { permitted } = useComponentPermissionState({
    permissions,
    project,
    checkBasicWorkspacePermissions: true,
  });

  useEffect(() => {
    onReady?.(permitted ? targetRef.current : null);
    return () => onReady?.(null);
  }, [onReady, permitted, project?.name, route.fullPath, routeKey]);

  if (!permitted) {
    return (
      <ComponentPermissionGuard
        permissions={permissions}
        project={project}
        className={className}
        path={route.fullPath}
        checkBasicWorkspacePermissions
        enableRequestRole
      >
        <div />
      </ComponentPermissionGuard>
    );
  }

  return <div ref={targetRef} className={targetClassName} />;
}
