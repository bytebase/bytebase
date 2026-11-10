import { ShieldUser } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FeatureBadge } from "@/components/FeatureBadge";
import { PermissionGuard } from "@/components/PermissionGuard";
import { Button } from "@/components/ui/button";
import { useAppProject } from "@/hooks/useAppProject";
import { cn } from "@/lib/utils";
import { RequestRoleSheet } from "@/modules/access-control/RequestRoleSheet";
import { getDefaultQueryRole } from "@/modules/sql-editor/model/requestQueryRole";
import { useSQLEditorEditorState } from "@/modules/sql-editor/store/editor";
import { useAppStore } from "@/stores/app";
import type { DatabaseResource, Permission } from "@/types";
import type { PermissionDeniedDetail } from "@/types/proto-es/v1/common_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { parseStringToResource } from "@/utils/v1/databaseResource";
import { AccessGrantRequestDrawer } from "./AccessGrantRequestDrawer";
import { useRequestDrawerHost } from "./RequestDrawerHost";

interface Props {
  readonly size?: "sm" | "default";
  readonly text?: boolean;
  readonly statement?: string;
  readonly permissionDeniedDetail: PermissionDeniedDetail;
}

export function RequestQueryButton({
  size = "default",
  text = false,
  statement,
  permissionDeniedDetail,
}: Props) {
  const { t } = useTranslation();
  const roleList = useAppStore((state) => state.roleList);

  // When the layout-level host is mounted (typical case inside the SQL
  // Editor), opening the drawer dispatches up to the host so it survives
  // ancestor unmounts (e.g. the connection panel Sheet closing). Local
  // state stays as a fallback for standalone callers (tests, isolated
  // pages without the host).
  const drawerHost = useRequestDrawerHost();
  const [showPanel, setShowPanel] = useState(false);
  const [showJITDrawer, setShowJITDrawer] = useState(false);

  const loadSubscription = useAppStore((s) => s.loadSubscription);
  // The SQL editor route doesn't mount the dashboard shells that load the
  // app-store subscription, so load it here before reading feature gates
  // (mirrors ComponentPermissionGuard).
  useEffect(() => {
    void loadSubscription();
  }, [loadSubscription]);

  const projectName = useSQLEditorEditorState((s) => s.project);
  const project = useAppProject(projectName);
  const hasCustomRoleFeature = useAppStore((s) =>
    s.hasInstanceFeature(PlanFeature.FEATURE_CUSTOM_ROLES)
  );
  const defaultQueryRole = useMemo(
    () =>
      getDefaultQueryRole(
        roleList,
        permissionDeniedDetail.requiredPermissions,
        hasCustomRoleFeature
      ),
    [roleList, permissionDeniedDetail.requiredPermissions, hasCustomRoleFeature]
  );

  const useJIT = useMemo(
    () =>
      !!project?.allowJustInTimeAccess &&
      permissionDeniedDetail.requiredPermissions.every(
        (perm) => perm === "bb.sql.select" || perm === "bb.sql.info"
      ),
    [project, permissionDeniedDetail.requiredPermissions]
  );

  const requiredPermission: Permission = useJIT
    ? "bb.accessGrants.create"
    : "bb.issues.create";

  const requiredFeature = useJIT
    ? PlanFeature.FEATURE_JIT
    : PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW;

  const hasRequestFeature = useAppStore((s) => s.hasFeature(requiredFeature));

  const missingResources = useMemo((): DatabaseResource[] => {
    const resources: DatabaseResource[] = [];
    for (const resourceString of permissionDeniedDetail.resources) {
      const resource = parseStringToResource(resourceString);
      if (resource) {
        resources.push(resource);
      }
    }
    return resources;
  }, [permissionDeniedDetail.resources]);

  const available = !!project?.allowRequestRole || useJIT;

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (useJIT) {
      if (drawerHost) {
        drawerHost.openAccessGrantDrawer({
          query: statement,
          targets: missingResources.map((r) => r.databaseFullName),
        });
      } else {
        setShowJITDrawer(true);
      }
    } else if (project) {
      if (drawerHost) {
        drawerHost.openRequestRoleSheet({
          project,
          requiredPermissions:
            permissionDeniedDetail.requiredPermissions as Permission[],
          initialRole: defaultQueryRole,
          initialDatabaseResources: missingResources,
        });
      } else {
        setShowPanel(true);
      }
    }
  };

  if (!available) {
    return null;
  }

  return (
    <div>
      <PermissionGuard permissions={[requiredPermission]} project={project}>
        {({ disabled }) => (
          <Button
            size={size}
            appearance={text ? "secondary" : "solid"}
            disabled={disabled || !hasRequestFeature}
            onClick={handleClick}
            className={cn(
              "gap-x-1",
              text && "text-accent hover:bg-transparent hover:text-accent-hover"
            )}
          >
            {hasRequestFeature ? (
              <ShieldUser className="size-4" />
            ) : (
              <FeatureBadge clickable={false} feature={requiredFeature} />
            )}
            {useJIT
              ? t("sql-editor.request-query")
              : t("issue.title.request-role")}
          </Button>
        )}
      </PermissionGuard>

      {showPanel && project && (
        <RequestRoleSheet
          open={showPanel}
          project={project}
          requiredPermissions={
            permissionDeniedDetail.requiredPermissions as Permission[]
          }
          initialRole={defaultQueryRole}
          initialDatabaseResources={missingResources}
          onClose={() => setShowPanel(false)}
        />
      )}

      {showJITDrawer && (
        <AccessGrantRequestDrawer
          query={statement}
          targets={missingResources.map((r) => r.databaseFullName)}
          onClose={() => setShowJITDrawer(false)}
        />
      )}
    </div>
  );
}
