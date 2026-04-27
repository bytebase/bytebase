import { ShieldAlert, ShieldUser } from "lucide-react";
import { lazy, type ReactNode, Suspense, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { REQUEST_ROLE_REQUIRED_PERMISSIONS } from "@/react/pages/settings/requestRoleButton";
import { usePermissionStore, useSubscriptionV1Store } from "@/store";
import { BASIC_WORKSPACE_PERMISSIONS, type Permission } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

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
  const permissionStore = usePermissionStore();
  const workspacePermissionKey = useVueState(() =>
    [...permissionStore.currentPermissions].sort().join("\n")
  );
  const projectPermissionKey = useVueState(() =>
    project
      ? [...permissionStore.currentPermissionsInProjectV1(project)]
          .sort()
          .join("\n")
      : ""
  );

  return useMemo(() => {
    const workspacePermissions = new Set(
      workspacePermissionKey.split("\n").filter(Boolean) as Permission[]
    );
    const projectPermissions = new Set(
      projectPermissionKey.split("\n").filter(Boolean) as Permission[]
    );
    const missedBasicPermissions = checkBasicWorkspacePermissions
      ? BASIC_WORKSPACE_PERMISSIONS.filter((p) => !workspacePermissions.has(p))
      : [];

    const missedPermissions = permissions.filter((p) =>
      project
        ? !workspacePermissions.has(p) && !projectPermissions.has(p)
        : !workspacePermissions.has(p)
    );

    return {
      missedBasicPermissions,
      missedPermissions,
      permitted:
        missedBasicPermissions.length === 0 && missedPermissions.length === 0,
    };
  }, [
    checkBasicWorkspacePermissions,
    permissions,
    project,
    projectPermissionKey,
    workspacePermissionKey,
  ]);
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
  const { t } = useTranslation();
  const [showRequestRoleSheet, setShowRequestRoleSheet] = useState(false);
  const subscriptionStore = useSubscriptionV1Store();
  const hasRequestRoleFeature = useVueState(() =>
    subscriptionStore.hasFeature(PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW)
  );
  const { missedBasicPermissions, missedPermissions, permitted } =
    useComponentPermissionState({
      permissions,
      project,
      checkBasicWorkspacePermissions,
    });

  if (permitted) {
    return <>{children}</>;
  }

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
              <PermissionGuard
                permissions={[...REQUEST_ROLE_REQUIRED_PERMISSIONS]}
                project={project}
              >
                {({ disabled }) => (
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={disabled || !hasRequestRoleFeature}
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
                )}
              </PermissionGuard>
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
