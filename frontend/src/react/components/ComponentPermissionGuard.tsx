import { ShieldAlert, ShieldUser } from "lucide-react";
import {
  lazy,
  type ReactNode,
  Suspense,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { Button } from "@/react/components/ui/button";
import { REQUEST_ROLE_REQUIRED_PERMISSIONS } from "@/react/pages/settings/requestRoleButton";
import { useAppStore } from "@/react/stores/app";
import { BASIC_WORKSPACE_PERMISSIONS, type Permission } from "@/types/iam";
import { hasFeature as planHasFeature } from "@/types/plan";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  PlanFeature,
  PlanType,
} from "@/types/proto-es/v1/subscription_service_pb";

const RequestRoleSheet = lazy(async () => {
  const module = await import("@/react/pages/settings/RequestRoleSheet");
  return { default: module.RequestRoleSheet };
});

interface ComponentPermissionGuardProps {
  readonly permissions: Permission[];
  readonly project?: Project;
  readonly children: ReactNode;
  readonly className?: string;
  readonly path?: string;
  readonly resources?: string[];
  readonly checkBasicWorkspacePermissions?: boolean;
  readonly enableRequestRole?: boolean;
}

interface ComponentPermissionState {
  missedBasicPermissions: Permission[];
  missedPermissions: Permission[];
  permitted: boolean;
}

interface PermissionDeniedFallbackProps {
  readonly missedBasicPermissions: Permission[];
  readonly missedPermissions: Permission[];
  readonly project?: Project;
  readonly className?: string;
  readonly path?: string;
  readonly resources?: string[];
  readonly enableRequestRole?: boolean;
}

export function usePermissionDataReady(project?: Project) {
  const permissionKey = project?.name ?? "__workspace__";
  const loadWorkspacePermissionState = useAppStore(
    (state) => state.loadWorkspacePermissionState
  );
  const loadProjectIamPolicy = useAppStore(
    (state) => state.loadProjectIamPolicy
  );
  const [readyKey, setReadyKey] = useState("");

  useEffect(() => {
    let stale = false;

    const requests: Promise<unknown>[] = [loadWorkspacePermissionState()];
    if (project?.name) {
      requests.push(loadProjectIamPolicy(project.name));
    }

    void Promise.all(requests).finally(() => {
      if (!stale) {
        setReadyKey(permissionKey);
      }
    });

    return () => {
      stale = true;
    };
  }, [loadProjectIamPolicy, loadWorkspacePermissionState, permissionKey]);

  return readyKey === permissionKey;
}

function usePermissionAccess(project?: Project) {
  const currentUserName = useAppStore((state) => state.currentUser?.name ?? "");
  const roles = useAppStore((state) => state.roles);
  const workspacePolicy = useAppStore((state) => state.workspacePolicy);
  const projectPolicy = useAppStore((state) =>
    project ? state.projectPoliciesByName[project.name] : undefined
  );
  const hasWorkspacePermission = useAppStore(
    (state) => state.hasWorkspacePermission
  );
  const hasProjectPermission = useAppStore(
    (state) => state.hasProjectPermission
  );

  return useMemo(
    () => ({
      hasRoutePermission: (permission: Permission) =>
        project
          ? hasProjectPermission(project, permission)
          : hasWorkspacePermission(permission),
      hasWorkspacePermission: (permission: Permission) =>
        hasWorkspacePermission(permission),
    }),
    [
      currentUserName,
      hasProjectPermission,
      hasWorkspacePermission,
      project,
      projectPolicy,
      roles,
      workspacePolicy,
    ]
  );
}

export function useComponentPermissionState({
  permissions,
  project,
  checkBasicWorkspacePermissions = false,
}: Readonly<
  Pick<
    ComponentPermissionGuardProps,
    "permissions" | "project" | "checkBasicWorkspacePermissions"
  >
>): ComponentPermissionState {
  const permissionAccess = usePermissionAccess(project);

  return useMemo(() => {
    const missedBasicPermissions = checkBasicWorkspacePermissions
      ? BASIC_WORKSPACE_PERMISSIONS.filter(
          (p) => !permissionAccess.hasWorkspacePermission(p)
        )
      : [];

    const missedPermissions = permissions.filter(
      (p) => !permissionAccess.hasRoutePermission(p)
    );

    return {
      missedBasicPermissions,
      missedPermissions,
      permitted:
        missedBasicPermissions.length === 0 && missedPermissions.length === 0,
    };
  }, [checkBasicWorkspacePermissions, permissionAccess, permissions]);
}

export function PermissionDeniedFallback({
  missedBasicPermissions,
  missedPermissions,
  project,
  className,
  path,
  resources = [],
  enableRequestRole = false,
}: PermissionDeniedFallbackProps) {
  const { t } = useTranslation();
  const [showRequestRoleSheet, setShowRequestRoleSheet] = useState(false);
  const loadSubscription = useAppStore((state) => state.loadSubscription);
  const subscriptionPlan = useAppStore(
    (state) => state.subscription?.plan ?? PlanType.FREE
  );
  const hasRequestRoleFeature = planHasFeature(
    subscriptionPlan,
    PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW
  );
  const requestRolePermissionAccess = usePermissionAccess(project);
  const canRequestRole = useMemo(
    () =>
      REQUEST_ROLE_REQUIRED_PERMISSIONS.every((permission) =>
        requestRolePermissionAccess.hasRoutePermission(permission)
      ),
    [requestRolePermissionAccess]
  );

  useEffect(() => {
    if (enableRequestRole) {
      void loadSubscription();
    }
  }, [enableRequestRole, loadSubscription]);

  const missed =
    missedBasicPermissions.length > 0
      ? missedBasicPermissions
      : missedPermissions;
  const showRequestRole =
    enableRequestRole &&
    missedBasicPermissions.length === 0 &&
    !!project &&
    project.allowRequestRole &&
    missedPermissions.length > 0;

  return (
    <div className={className}>
      <div
        role="alert"
        className="relative w-full rounded-xs border border-error/30 bg-error/5 text-error px-4 py-3 text-sm flex gap-x-3"
      >
        <ShieldAlert className="size-5 shrink-0 mt-0.5" />
        <div className="flex flex-col gap-3">
          <h5 className="font-medium leading-tight">
            {project
              ? t("common.missing-required-permission-for-resource", {
                  resource: project.name,
                })
              : t("common.missing-required-permission", { permissions: "" })}
          </h5>
          <div>
            {t("common.required-permission")}
            <ul className="list-disc pl-4">
              {missed.map((p) => (
                <li key={p}>{p}</li>
              ))}
            </ul>
          </div>
          {(path || resources.length > 0) && (
            <div className="text-error/80">
              {path && <div>{path}</div>}
              {resources.map((resource) => (
                <div key={resource}>{resource}</div>
              ))}
            </div>
          )}
          {showRequestRole && (
            <div>
              <Button
                variant="outline"
                size="sm"
                disabled={!canRequestRole || !hasRequestRoleFeature}
                onClick={() => setShowRequestRoleSheet(true)}
              >
                {hasRequestRoleFeature ? (
                  <ShieldUser className="size-4" />
                ) : (
                  <FeatureBadge
                    feature={PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW}
                    clickable={false}
                  />
                )}
                {t("issue.title.request-role")}
              </Button>
            </div>
          )}
        </div>
      </div>
      {project && showRequestRoleSheet && (
        <Suspense fallback={null}>
          <RequestRoleSheet
            open={showRequestRoleSheet}
            project={project}
            requiredPermissions={missedPermissions}
            onClose={() => setShowRequestRoleSheet(false)}
          />
        </Suspense>
      )}
    </div>
  );
}

/**
 * ComponentPermissionGuard gates an entire component behind a permission check.
 *
 * - If the user has all required permissions, children are rendered normally.
 * - If the user is missing permissions, an error alert is shown listing the
 *   missing permissions — matching the Vue `ComponentPermissionGuard` behavior.
 */
export function ComponentPermissionGuard({
  permissions,
  project,
  children,
  className,
  path,
  resources = [],
  checkBasicWorkspacePermissions = false,
  enableRequestRole = false,
}: ComponentPermissionGuardProps) {
  const permissionReady = usePermissionDataReady(project);
  const { missedBasicPermissions, missedPermissions, permitted } =
    useComponentPermissionState({
      permissions,
      project,
      checkBasicWorkspacePermissions,
    });

  if (!permissionReady) {
    return <div className={className} />;
  }

  if (permitted) {
    return <>{children}</>;
  }

  return (
    <PermissionDeniedFallback
      missedBasicPermissions={missedBasicPermissions}
      missedPermissions={missedPermissions}
      project={project}
      className={className}
      path={path}
      resources={resources}
      enableRequestRole={enableRequestRole}
    />
  );
}
