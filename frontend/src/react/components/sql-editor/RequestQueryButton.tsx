import { ShieldUser } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { RequestRoleSheet } from "@/react/pages/settings/RequestRoleSheet";
import { useSQLEditorVueState } from "@/react/stores/sqlEditor/editor-vue-state";
import {
  hasFeature,
  useProjectV1Store,
  useRoleStore,
  useSubscriptionV1Store,
} from "@/store";
import type { DatabaseResource, Permission } from "@/types";
import { PRESET_ROLES, PresetRoleType } from "@/types";
import type { PermissionDeniedDetail } from "@/types/proto-es/v1/common_pb";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { parseStringToResource } from "@/utils/v1/databaseResource";
import { AccessGrantRequestDrawer } from "./AccessGrantRequestDrawer";
import { useRequestDrawerHost } from "./RequestDrawerHost";

const SQL_SELECT_PERMISSION = "bb.sql.select";

const getDefaultQueryRole = (
  roles: readonly Pick<Role, "name" | "permissions">[],
  requiredPermissions: readonly string[],
  hasCustomRoleFeature: boolean
) => {
  const permissions =
    requiredPermissions.length > 0
      ? requiredPermissions
      : [SQL_SELECT_PERMISSION];
  const candidates = [...roles]
    .filter((role) => hasCustomRoleFeature || PRESET_ROLES.includes(role.name))
    .filter((role) =>
      permissions.every((permission) => role.permissions.includes(permission))
    )
    .sort((a, b) => {
      const permissionCountDelta = a.permissions.length - b.permissions.length;
      if (permissionCountDelta !== 0) {
        return permissionCountDelta;
      }
      if (a.name === PresetRoleType.SQL_EDITOR_READ_USER) {
        return -1;
      }
      if (b.name === PresetRoleType.SQL_EDITOR_READ_USER) {
        return 1;
      }
      return a.name.localeCompare(b.name);
    });

  return candidates[0]?.name ?? PresetRoleType.SQL_EDITOR_USER;
};

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
  const editorStore = useSQLEditorVueState();
  const roleStore = useRoleStore();
  const subscriptionStore = useSubscriptionV1Store();

  const projectName = useVueState(() => editorStore.project);
  const project = useVueState(() => projectStore.getProjectByName(projectName));
  const hasCustomRoleFeature = useVueState(() =>
    subscriptionStore.hasInstanceFeature(PlanFeature.FEATURE_CUSTOM_ROLES)
  );
  const defaultQueryRole = useVueState(() =>
    getDefaultQueryRole(
      roleStore.roleList,
      permissionDeniedDetail.requiredPermissions,
      hasCustomRoleFeature
    )
  );

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
