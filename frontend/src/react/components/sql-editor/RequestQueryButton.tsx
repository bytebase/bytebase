import { ShieldUser } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { parseStringToResource } from "@/components/RoleGrantPanel/DatabaseResourceForm/common";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { RequestRoleSheet } from "@/react/pages/settings/RequestRoleSheet";
import { hasFeature, useProjectV1Store, useSQLEditorStore } from "@/store";
import type { DatabaseResource, Permission } from "@/types";
import { PresetRoleType } from "@/types";
import type { PermissionDeniedDetail } from "@/types/proto-es/v1/common_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
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

  // When the layout-level host is mounted (typical case inside the SQL
  // Editor), opening the drawer dispatches up to the host so it survives
  // ancestor unmounts (e.g. the connection panel Sheet closing). Local
  // state stays as a fallback for standalone callers (tests, isolated
  // pages without the host).
  const drawerHost = useRequestDrawerHost();
  const [showPanel, setShowPanel] = useState(false);
  const [showJITDrawer, setShowJITDrawer] = useState(false);

  const projectStore = useProjectV1Store();
  const editorStore = useSQLEditorStore();

  const projectName = useVueState(() => editorStore.project);
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  const useJIT = useMemo(
    () =>
      !!project?.allowJustInTimeAccess &&
      permissionDeniedDetail.requiredPermissions.every(
        (perm) => perm === "bb.sql.select"
      ),
    [project, permissionDeniedDetail.requiredPermissions]
  );

  const requiredPermission: Permission = useJIT
    ? "bb.accessGrants.create"
    : "bb.issues.create";

  const requiredFeature = useJIT
    ? PlanFeature.FEATURE_JIT
    : PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW;

  const hasRequestFeature = hasFeature(requiredFeature);

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

  const available = useMemo(() => !!project?.allowRequestRole, [project]);

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
          initialRole: PresetRoleType.SQL_EDITOR_USER,
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
            variant={text ? "ghost" : "default"}
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
              ? t("sql-editor.request-jit")
              : t("sql-editor.request-query")}
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
          initialRole={PresetRoleType.SQL_EDITOR_USER}
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
