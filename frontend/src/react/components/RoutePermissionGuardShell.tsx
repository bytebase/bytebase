import { useEffect, useRef } from "react";
import {
  ComponentPermissionGuard,
  useComponentPermissionState,
  usePermissionDataReady,
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
  const onReadyRef = useRef(onReady);
  const permissions = route.requiredPermissions;
  const permissionReady = usePermissionDataReady(project);
  const { permitted } = useComponentPermissionState({
    permissions,
    project,
    checkBasicWorkspacePermissions: true,
  });

  useEffect(() => {
    onReadyRef.current = onReady;
  }, [onReady]);

  useEffect(() => {
    onReadyRef.current?.(null);
    return () => {
      onReadyRef.current?.(null);
    };
  }, [project?.name, route.fullPath, routeKey]);

  useEffect(() => {
    if (!permissionReady) {
      onReadyRef.current?.(null);
      return;
    }
    onReadyRef.current?.(permitted ? targetRef.current : null);
  }, [permissionReady, permitted, project?.name, route.fullPath, routeKey]);

  if (!permissionReady) {
    return <div className={targetClassName} />;
  }

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
