import { ArrowRightLeft, Pencil } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { preCreateIssue } from "@/react/lib/plan/issue";
import { useAppStore } from "@/react/stores/app";
import type { Permission } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { DatabaseExportSchemaButton } from "./DatabaseExportSchemaButton";
import { DatabaseSyncButton } from "./DatabaseSyncButton";

const DATABASE_CHANGE_PERMISSIONS: Permission[] = [
  "bb.plans.create",
  "bb.sheets.create",
];

export function DatabaseDetailActions({
  database,
  isDefaultProject,
  onOpenTransferProject,
}: {
  database: Database;
  isDefaultProject: boolean;
  onOpenTransferProject: () => void;
}) {
  const { t } = useTranslation();
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);
  const hasProjectPermissionFn = useAppStore(
    (state) => state.hasProjectPermission
  );
  const hasWorkspacePermissionFn = useAppStore(
    (state) => state.hasWorkspacePermission
  );
  const project = useVueState(() =>
    useAppStore.getState().getProjectByName(database.project)
  );
  void projectsByName;
  const hasProjectPermission = useMemo(
    () => (permission: Permission) => {
      if (hasWorkspacePermissionFn(permission)) {
        return true;
      }
      return project ? hasProjectPermissionFn(project, permission) : false;
    },
    [hasProjectPermissionFn, hasWorkspacePermissionFn, project]
  );

  const canUpdate = useMemo(
    () => hasProjectPermission("bb.databases.update"),
    [hasProjectPermission]
  );
  const canChangeDatabase = useMemo(
    () =>
      DATABASE_CHANGE_PERMISSIONS.every((permission) =>
        hasProjectPermission(permission)
      ),
    [hasProjectPermission]
  );
  const canSync = useMemo(
    () => hasProjectPermission("bb.databases.sync"),
    [hasProjectPermission]
  );
  const canExportSchema = useMemo(
    () => hasProjectPermission("bb.databases.getSchema"),
    [hasProjectPermission]
  );

  return (
    <>
      <div className="flex shrink-0 flex-wrap items-center justify-start gap-x-2 gap-y-2">
        <PermissionGuard permissions={["bb.databases.sync"]} project={project}>
          <DatabaseSyncButton database={database} disabled={!canSync} />
        </PermissionGuard>
        <PermissionGuard
          permissions={["bb.databases.getSchema"]}
          project={project}
        >
          <DatabaseExportSchemaButton
            database={database}
            disabled={!canExportSchema}
          />
        </PermissionGuard>
        {!isDefaultProject && (
          <PermissionGuard
            permissions={["bb.databases.update"]}
            project={project}
          >
            <Button
              variant="outline"
              disabled={!canUpdate}
              onClick={onOpenTransferProject}
            >
              <ArrowRightLeft className="h-4 w-4" />
              {t("database.transfer-project")}
            </Button>
          </PermissionGuard>
        )}
        {!isDefaultProject && (
          <PermissionGuard
            permissions={["bb.plans.create", "bb.sheets.create"]}
            project={project}
          >
            <Button
              variant="outline"
              disabled={!canChangeDatabase}
              onClick={() =>
                void preCreateIssue(database.project, [database.name])
              }
            >
              <Pencil className="h-4 w-4" />
              {t("database.change-database")}
            </Button>
          </PermissionGuard>
        )}
      </div>
    </>
  );
}
